package phonedoor_test

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/phonedoor"
	"github.com/codedogapp/nudge/internal/sessions"
	"github.com/codedogapp/nudge/internal/subscriptions"
)

// The user agent's keys of RFC 8291, Appendix A.
const (
	p256dh = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	auth   = "BTBZMqHH6r4Tts7J_aSIgg"
)

// browser is a subscription as a browser's PushSubscription.toJSON() gives it.
const browser = `{"endpoint": "https://push.example/send/1", "expirationTime": null, ` +
	`"keys": {"p256dh": "` + p256dh + `", "auth": "` + auth + `"}}`

// subscribing returns a same-origin POST /subscriptions with body.
func subscribing(t *testing.T, body string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/subscriptions", strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	return req
}

// devicesAt returns a store that keeps the push subscriptions in the file at path.
func devicesAt(t *testing.T, path string) *subscriptions.Store {
	t.Helper()

	store, err := subscriptions.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

// subscribeTo serves req on a door that keeps the subscriptions in the file at path.
func subscribeTo(t *testing.T, path string, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	return subscribe(t, devicesAt(t, path), req)
}

// subscribe serves req on a door that adds the subscriptions to store.
func subscribe(
	t *testing.T, store *subscriptions.Store, req *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	var box inbox.Inbox

	rec := httptest.NewRecorder()
	phonedoor.Routes(&box, new(sessions.Registry), pushKey, store).ServeHTTP(rec, req)

	return rec
}

func folderOf(t *testing.T, path string) *os.Root {
	t.Helper()

	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = root.Close() })

	return root
}

func assertNoFile(t *testing.T, path string) {
	t.Helper()

	if _, err := folderOf(t, path).Stat(filepath.Base(path)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(the subscriptions file) = %v, want it missing", err)
	}
}

func TestSubscribeKeepsTheSubscription(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscriptions.json")

	rec := subscribeTo(t, path, subscribing(t, browser))
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("POST = %d %q, want 204 and no body", rec.Code, rec.Body)
	}

	root := folderOf(t, path)

	info, err := root.Stat(filepath.Base(path))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("Stat(the file) = %v, %v; want mode 0600", info, err)
	}

	data, err := root.ReadFile(filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}

	var got []subscriptions.Subscription
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	want := []subscriptions.Subscription{{
		Endpoint: "https://push.example/send/1",
		Keys:     subscriptions.Keys{P256dh: p256dh, Auth: auth},
	}}
	if !slices.Equal(got, want) {
		t.Errorf("the file holds %+v, want %+v", got, want)
	}
}

func TestSubscribeRefusesABadBody(t *testing.T) {
	long := strings.Replace(browser, "send/1", "send/"+strings.Repeat("1", 4<<10), 1)

	tests := []struct{ name, body, reason string }{
		{"not JSON", "{", "read the subscription: "},
		{
			"an invalid subscription",
			strings.Replace(browser, "https://", "http://", 1),
			"invalid subscription: the endpoint",
		},
		{"a body over 4 KiB", long, "request body too large"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "subscriptions.json")

			rec := subscribeTo(t, path, subscribing(t, test.body))
			if rec.Code != http.StatusBadRequest ||
				!strings.Contains(rec.Body.String(), test.reason) {
				t.Errorf("POST = %d %q, want 400 and %q", rec.Code, rec.Body, test.reason)
			}

			assertNoFile(t, path)
		})
	}
}

func TestSubscribeReportsAFailedWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subscriptions.json")
	store := devicesAt(t, path)

	// A file in the folder's place makes the write fail.
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(dir, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	rec := subscribe(t, store, subscribing(t, browser))
	if want := "write the subscriptions file " + path; rec.Code != http.StatusInternalServerError ||
		!strings.HasPrefix(rec.Body.String(), want) {
		t.Errorf("POST = %d %q, want 500 and %q", rec.Code, rec.Body, want)
	}
}

func TestCrossOriginSubscriptionIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscriptions.json")

	req := subscribing(t, browser)
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	if rec := subscribeTo(t, path, req); rec.Code != http.StatusForbidden {
		t.Errorf("POST = %d, want 403", rec.Code)
	}

	assertNoFile(t, path)
}
