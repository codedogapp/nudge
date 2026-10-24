package agentdoor_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"uuid"

	"github.com/codedogapp/nudge/internal/inbox"
)

const merge = `{"project":"nudge","text":"Merge?","details":"CI is green.","options":["Yes","No"]}`

// only returns the one question that waits in the inbox.
func only(t *testing.T, box *inbox.Inbox) inbox.Question {
	t.Helper()

	pending := box.Pending()
	if len(pending) != 1 {
		t.Fatalf("Pending() = %+v, want one question", pending)
	}

	return pending[0]
}

func TestAnswerGoesBackToTheAgent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		done := agent.ask(t.Context())

		waiting := only(t, &agent.box)
		asked := waiting
		asked.ID = uuid.Nil()
		want := inbox.Question{
			Project: "nudge",
			Text:    "Merge?",
			Details: "CI is green.",
			Options: []string{"Yes", "No"},
		}

		if !reflect.DeepEqual(asked, want) {
			t.Errorf("the waiting question without its ID = %+v, want %+v", asked, want)
		}

		err := agent.box.Answer(waiting.ID, inbox.Answer{Option: "Yes", Text: "squash"})
		if err != nil {
			t.Fatalf("Answer() = %v, want nil", err)
		}

		got := <-done
		if got.err != nil || got.status != http.StatusOK {
			t.Fatalf("reply = %d, %v, want 200", got.status, got.err)
		}

		if kind := got.header.Get("Content-Type"); kind != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", kind)
		}

		if want := `{"option":"Yes","text":"squash"}`; got.body != want {
			t.Errorf("body = %s, want %s", got.body, want)
		}
	})
}

func TestReplyKeepsInvalidUTF8Out(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		done := agent.ask(t.Context())

		err := agent.box.Answer(only(t, &agent.box).ID, inbox.Answer{Text: "ok\xff"})
		if err != nil {
			t.Fatalf("Answer() = %v, want nil", err)
		}

		if got, want := <-done, "{\"option\":\"\",\"text\":\"ok\ufffd\"}"; got.body != want {
			t.Errorf("reply = %d %s, %v, want 200 %s", got.status, got.body, got.err, want)
		}
	})
}

func TestQuestionLeavesWithItsAgent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)

		ctx, cancel := context.WithCancel(t.Context())
		done := agent.ask(ctx)
		only(t, &agent.box)

		cancel()
		synctest.Wait()

		if pending := agent.box.Pending(); len(pending) != 0 {
			t.Errorf("Pending() = %+v after the agent left, want none", pending)
		}

		if got := <-done; !errors.Is(got.err, context.Canceled) {
			t.Errorf("reply error = %v, want %v", got.err, context.Canceled)
		}
	})
}

func TestStoppingEndsTheQuestionsThatWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, stop := context.WithCancel(t.Context())
		agent := serve(ctx, t)
		done := agent.ask(t.Context())
		only(t, &agent.box)

		stop()

		got := <-done
		if got.err != nil || got.status != http.StatusServiceUnavailable {
			t.Errorf("reply = %d, %v, want 503", got.status, got.err)
		}

		if want := "nudge serve is stopping\n"; got.body != want {
			t.Errorf("body = %q, want %q", got.body, want)
		}

		if pending := agent.box.Pending(); len(pending) != 0 {
			t.Errorf("Pending() = %+v after the stop, want none", pending)
		}
	})
}

func TestDoorRefuses(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"not JSON", http.MethodPost, "/questions", "Merge?", http.StatusBadRequest},
		{
			"unknown field",
			http.MethodPost,
			"/questions",
			`{"text":"Merge?","colour":"red"}`,
			http.StatusBadRequest,
		},
		{
			"invalid UTF-8",
			http.MethodPost,
			"/questions",
			"{\"text\":\"\xff\"}",
			http.StatusBadRequest,
		},
		{
			"over 64 KiB",
			http.MethodPost,
			"/questions",
			`{"text":"` + strings.Repeat("a", 64<<10) + `"}`,
			http.StatusBadRequest,
		},
		{"no text", http.MethodPost, "/questions", `{"project":"nudge"}`, http.StatusBadRequest},
		{
			"repeated option",
			http.MethodPost,
			"/questions",
			`{"text":"Merge?","options":["Yes","Yes"]}`,
			http.StatusBadRequest,
		},
		{"other method", http.MethodGet, "/questions", "", http.StatusMethodNotAllowed},
		{"other path", http.MethodPost, "/answers", merge, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// In a bubble, a request that waits instead of failing fails the test as a deadlock.
			synctest.Test(t, func(t *testing.T) {
				agent := serve(t.Context(), t)

				got := agent.do(t.Context(), test.method, test.path, test.body)
				if got.err != nil || got.status != test.want {
					t.Errorf("reply = %d, %v, want %d", got.status, got.err, test.want)
				}

				if pending := agent.box.Pending(); len(pending) != 0 {
					t.Errorf("Pending() = %+v, want none", pending)
				}
			})
		})
	}
}
