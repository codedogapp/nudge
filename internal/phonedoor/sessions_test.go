package phonedoor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/phonedoor"
	"github.com/codedogapp/nudge/internal/sessions"
)

// pageWith returns the page's HTML as served with the sessions in live.
func pageWith(t *testing.T, live *sessions.Registry) string {
	t.Helper()

	rec := httptest.NewRecorder()
	phonedoor.Routes(new(inbox.Inbox), live, pushKey, devices(t)).
		ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	return rec.Body.String()
}

func TestPageListsTheSessionsWithTheirNotes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var live sessions.Registry

		live.Start("bakhyard")
		noted, _ := live.Start("nudge")

		if err := live.Report(noted.ID, "tests are green"); err != nil {
			t.Fatal(err)
		}

		body := pageWith(t, &live)
		want := []string{
			"Sessions",
			"nudge",
			"tests are green",
			"just now",
			"bakhyard",
			"No report yet.",
		}
		offset := 0

		for _, text := range want {
			found := strings.Index(body[offset:], text)
			if found < 0 {
				t.Fatalf("page = %s, want %q in this order", body, want)
			}

			offset += found + len(text)
		}
	})
}

func TestPageSaysWhenNoSessionRuns(t *testing.T) {
	if body := pageWith(
		t,
		new(sessions.Registry),
	); !strings.Contains(
		body,
		"No agent session is running.",
	) {
		t.Errorf("page = %s, want it to say no session runs", body)
	}
}

func TestPageEscapesTheNotes(t *testing.T) {
	var live sessions.Registry

	started, _ := live.Start("<i>nudge</i>")

	if err := live.Report(started.ID, "<script>alert(1)</script>"); err != nil {
		t.Fatal(err)
	}

	body := pageWith(t, &live)

	for _, raw := range []string{"<i>", "<script>"} {
		if strings.Contains(body, raw) {
			t.Errorf("page = %s, want %q escaped", body, raw)
		}
	}
}

func TestPageShowsHowOldANoteIs(t *testing.T) {
	tests := []struct {
		after time.Duration
		want  string
	}{
		{59 * time.Second, "just now"},
		{time.Minute, "1 min ago"},
		{59 * time.Minute, "59 min ago"},
		{time.Hour, "1 h ago"},
		{23*time.Hour + 59*time.Minute, "23 h ago"},
		{24 * time.Hour, "1 d ago"},
		{72 * time.Hour, "3 d ago"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var live sessions.Registry

				started, _ := live.Start("nudge")

				if err := live.Report(started.ID, "tests are green"); err != nil {
					t.Fatal(err)
				}

				time.Sleep(test.after)

				if body := pageWith(t, &live); !strings.Contains(body, test.want) {
					t.Errorf("page = %s, want %q", body, test.want)
				}
			})
		})
	}
}
