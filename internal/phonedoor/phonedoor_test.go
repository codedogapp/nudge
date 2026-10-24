package phonedoor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"uuid"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/phonedoor"
)

// waiting returns an inbox where questions wait, asked in their order, until the test ends.
func waiting(t *testing.T, questions ...inbox.Question) *inbox.Inbox {
	t.Helper()

	var box inbox.Inbox

	for _, question := range questions {
		go func() { _, _ = box.Ask(t.Context(), question) }()

		synctest.Wait()
	}

	return &box
}

func get(t *testing.T, box *inbox.Inbox, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	phonedoor.Routes(box).
		ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))

	return rec
}

func TestPageListsTheQuestionsNewestFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := waiting(t,
			inbox.Question{
				Project: "nudge", Text: "Merge?", Details: "CI is green.",
				Options: []string{"Squash", "Rebase"},
			},
			inbox.Question{Project: "bakhyard", Text: "Deploy?"},
		)

		rec := get(t, box, http.MethodGet, "/")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET / = %d, want 200", rec.Code)
		}

		if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/html; charset=utf-8", got)
		}

		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q, want no-store", got)
		}

		body := rec.Body.String()
		want := []string{
			"bakhyard",
			"Deploy?",
			"nudge",
			"Merge?",
			"CI is green.",
			"Squash",
			"Rebase",
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

func TestPageEscapesTheQuestions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := waiting(t, inbox.Question{
			Project: "<i>nudge</i>",
			Text:    "<script>alert(1)</script>",
			Details: `"a" & 'b'`,
			Options: []string{"<b>Yes</b>", `"No"`},
		})

		body := get(t, box, http.MethodGet, "/").Body.String()

		for _, raw := range []string{"<i>", "<script>", "<b>", `"a"`, " & "} {
			if strings.Contains(body, raw) {
				t.Errorf("page = %s, want %q escaped", body, raw)
			}
		}

		for _, escaped := range []string{
			"&lt;i&gt;nudge&lt;/i&gt;",
			"&lt;script&gt;alert(1)&lt;/script&gt;",
			"&#34;a&#34; &amp; &#39;b&#39;",
			"&lt;b&gt;Yes&lt;/b&gt;",
			`value="&#34;No&#34;"`,
		} {
			if !strings.Contains(body, escaped) {
				t.Errorf("page = %s, want it to contain %q", body, escaped)
			}
		}
	})
}

func TestPageHasAFormPerQuestion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := waiting(t,
			inbox.Question{Text: "Merge?", Options: []string{"Squash", "Rebase"}},
			inbox.Question{Text: "Deploy?"},
		)

		body := get(t, box, http.MethodGet, "/").Body.String()

		for _, question := range box.Pending() {
			want := `<form method="post" action="/questions/` + question.ID.String() + `/answer">`
			if !strings.Contains(body, want) {
				t.Errorf("page = %s, want it to contain %s", body, want)
			}
		}

		for _, want := range []string{
			`<button name="option" value="Squash" formnovalidate>Squash</button>`,
			`<button name="option" value="Rebase" formnovalidate>Rebase</button>`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("page = %s, want it to contain %s", body, want)
			}
		}

		for _, want := range []string{
			`<textarea name="text" required></textarea>`,
			`<button>Send the reply</button>`,
		} {
			if got := strings.Count(body, want); got != len(box.Pending()) {
				t.Errorf("page = %s, want %s once per question, got %d", body, want, got)
			}
		}
	})
}

func TestPageSaysWhenNoQuestionWaits(t *testing.T) {
	var box inbox.Inbox

	rec := get(t, &box, http.MethodGet, "/")
	if want := "No question waits for an answer."; !strings.Contains(rec.Body.String(), want) {
		t.Errorf("GET / = %d %s, want it to say %q", rec.Code, rec.Body, want)
	}
}

func TestDoorServesOnlyItsRoutes(t *testing.T) {
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodHead, "/", http.StatusOK},
		{http.MethodPost, "/", http.StatusMethodNotAllowed},
		{http.MethodGet, "/questions", http.StatusNotFound},
		{
			http.MethodGet,
			"/questions/" + uuid.New().String() + "/answer",
			http.StatusMethodNotAllowed,
		},
	}

	for _, test := range tests {
		var box inbox.Inbox

		if got := get(t, &box, test.method, test.path).Code; got != test.want {
			t.Errorf("%s %s = %d, want %d", test.method, test.path, got, test.want)
		}
	}
}
