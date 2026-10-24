package phonedoor_test

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestDoorServesTheLook(t *testing.T) {
	tests := []struct {
		path, file, contentType string
	}{
		{"/style.css", "style.css", "text/css; charset=utf-8"},
		{"/grain.svg", "grain.svg", "image/svg+xml"},
	}

	for _, test := range tests {
		want, err := os.ReadFile(test.file)
		if err != nil {
			t.Fatal(err)
		}

		var box inbox.Inbox

		rec := get(t, &box, http.MethodGet, test.path)
		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), want) {
			t.Errorf("GET %s = %d with %d bytes, want 200 with the %d bytes of %s",
				test.path, rec.Code, rec.Body.Len(), len(want), test.file)
		}

		for name, value := range map[string]string{
			"Content-Type":  test.contentType,
			"Cache-Control": "no-cache",
		} {
			if got := rec.Header().Get(name); got != value {
				t.Errorf("GET %s: %s = %q, want %q", test.path, name, got, value)
			}
		}
	}
}

func TestDoorServesNoOtherFile(t *testing.T) {
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodHead, "/style.css", http.StatusOK},
		{http.MethodPost, "/style.css", http.StatusMethodNotAllowed},
		{http.MethodPost, "/grain.svg", http.StatusMethodNotAllowed},
		{http.MethodGet, "/style.css/", http.StatusNotFound},
		{http.MethodGet, "/page.html", http.StatusNotFound},
		{http.MethodGet, "/assets.go", http.StatusNotFound},
	}

	for _, test := range tests {
		var box inbox.Inbox

		if got := get(t, &box, test.method, test.path).Code; got != test.want {
			t.Errorf("%s %s = %d, want %d", test.method, test.path, got, test.want)
		}
	}
}

func TestPageLinksTheStylesheet(t *testing.T) {
	var box inbox.Inbox

	body := get(t, &box, http.MethodGet, "/").Body.String()
	if want := `<link rel="stylesheet" href="/style.css">`; !strings.Contains(body, want) {
		t.Errorf("page = %s, want it to contain %s", body, want)
	}
}
