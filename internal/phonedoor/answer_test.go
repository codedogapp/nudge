package phonedoor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"
	"uuid"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/phonedoor"
	"github.com/codedogapp/nudge/internal/sessions"
)

const gone = "This question has gone: it was answered, or its agent stopped waiting."

// asked returns an inbox where the question waits until the test ends, and a channel that gets
// its answer.
func asked(t *testing.T, question inbox.Question) (*inbox.Inbox, <-chan inbox.Answer) {
	t.Helper()

	var box inbox.Inbox

	answers := make(chan inbox.Answer, 1)

	go func() {
		if answer, err := box.Ask(t.Context(), question); err == nil {
			answers <- answer
		}
	}()

	synctest.Wait()

	return &box, answers
}

// answered returns the answer that the asker got, once the inbox is quiet.
func answered(t *testing.T, answers <-chan inbox.Answer) inbox.Answer {
	t.Helper()
	synctest.Wait()

	select {
	case got := <-answers:
		return got
	default:
		t.Fatal("the asker got no answer")

		return inbox.Answer{}
	}
}

// answerPath is the answer URL of the question that waits in box.
func answerPath(box *inbox.Inbox) string {
	return "/questions/" + box.Pending()[0].ID.String() + "/answer"
}

// form is a POST of the form to path, as a browser on the page's own origin sends it.
func form(t *testing.T, path string, values url.Values) *http.Request {
	t.Helper()

	body := strings.NewReader(values.Encode())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	return req
}

func serve(t *testing.T, box *inbox.Inbox, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	phonedoor.Routes(box, new(sessions.Registry), pushKey, devices(t)).ServeHTTP(rec, req)

	return rec
}

func TestAnswerReachesTheAsker(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
		want   inbox.Answer
	}{
		{
			"an option",
			url.Values{"option": {"Squash"}, "text": {""}},
			inbox.Answer{Option: "Squash"},
		},
		{"a reply", url.Values{"text": {" Wait for CI. "}}, inbox.Answer{Text: "Wait for CI."}},
		{
			"both",
			url.Values{"option": {"Rebase"}, "text": {"Then deploy."}},
			inbox.Answer{Option: "Rebase", Text: "Then deploy."},
		},
		{
			"a reply of several lines",
			url.Values{"text": {"One.\r\nTwo.\r\n\r\nThree."}},
			inbox.Answer{Text: "One.\nTwo.\n\nThree."},
		},
		{
			"an option of several lines",
			url.Values{"option": {"Ship\r\nit"}},
			inbox.Answer{Option: "Ship\nit"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				box, answers := asked(t, inbox.Question{
					Text: "Merge?", Options: []string{"Squash", "Rebase", "Ship\nit"},
				})

				rec := serve(t, box, form(t, answerPath(box), test.values))
				if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
					t.Fatalf(
						"POST = %d to %q, want 303 to /",
						rec.Code,
						rec.Header().Get("Location"),
					)
				}

				if got := answered(t, answers); got != test.want {
					t.Errorf("answer = %+v, want %+v", got, test.want)
				}

				if pending := box.Pending(); len(pending) != 0 {
					t.Errorf("pending = %+v, want none", pending)
				}
			})
		})
	}
}

func TestAnswerIsRefused(t *testing.T) {
	tests := []struct {
		name   string
		path   func(box *inbox.Inbox) string
		values url.Values
		status int
		reason string
	}{
		{
			"empty",
			answerPath,
			url.Values{"option": {""}, "text": {" \r\n "}},
			http.StatusBadRequest,
			"The answer was refused: the answer has neither an option nor a text.",
		},
		{
			"an option the question doesn't offer", answerPath,
			url.Values{"option": {"Merge"}},
			http.StatusBadRequest, "The answer was refused: the question doesn&#39;t offer",
		},
		{
			"too long", answerPath,
			url.Values{"text": {strings.Repeat("a", 64<<10)}},
			http.StatusBadRequest, "The answer was refused: http: request body too large.",
		},
		{
			"to another question",
			func(*inbox.Inbox) string { return "/questions/" + uuid.New().String() + "/answer" },
			url.Values{"option": {"Squash"}},
			http.StatusNotFound, gone,
		},
		{
			"to an ID that isn't one",
			func(*inbox.Inbox) string { return "/questions/42/answer" },
			url.Values{"option": {"Squash"}},
			http.StatusNotFound, gone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				box, _ := asked(t, inbox.Question{Text: "Merge?", Options: []string{"Squash"}})

				rec := serve(t, box, form(t, test.path(box), test.values))
				if rec.Code != test.status {
					t.Errorf("POST = %d, want %d", rec.Code, test.status)
				}

				body := rec.Body.String()
				if !strings.Contains(body, test.reason) || !strings.Contains(body, "Merge?") {
					t.Errorf("page = %s, want %q and the question that waits", body, test.reason)
				}

				if pending := box.Pending(); len(pending) != 1 {
					t.Errorf("pending = %+v, want the question to keep waiting", pending)
				}
			})
		})
	}
}

func TestAnswerToAnAnsweredQuestionIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box, _ := asked(t, inbox.Question{Text: "Merge?"})
		path := answerPath(box)

		if rec := serve(
			t,
			box,
			form(t, path, url.Values{"text": {"Yes."}}),
		); rec.Code != http.StatusSeeOther {
			t.Fatalf("first POST = %d, want 303", rec.Code)
		}

		rec := serve(t, box, form(t, path, url.Values{"text": {"No."}}))
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), gone) {
			t.Errorf("second POST = %d %s, want 404 and %q", rec.Code, rec.Body, gone)
		}
	})
}
