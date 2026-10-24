package inbox_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"uuid"

	"github.com/codedogapp/nudge/internal/inbox"
)

type result struct {
	answer inbox.Answer
	err    error
}

// ask asks in a goroutine of the test's bubble, and returns once the question waits in the inbox.
func ask(ctx context.Context, box *inbox.Inbox, question inbox.Question) <-chan result {
	done := make(chan result, 1)

	go func() {
		answer, err := box.Ask(ctx, question)
		done <- result{answer: answer, err: err}
	}()

	synctest.Wait()

	return done
}

// only returns the one question that waits in the inbox.
func only(t *testing.T, box *inbox.Inbox) inbox.Question {
	t.Helper()

	pending := box.Pending()
	if len(pending) != 1 {
		t.Fatalf("Pending() = %+v, want one question", pending)
	}

	return pending[0]
}

func merge() inbox.Question {
	return inbox.Question{
		Project: "nudge",
		Text:    "Merge the pull request?",
		Details: "CI is green.",
		Options: []string{"Yes", "No"},
	}
}

func TestAnswerReachesTheAsker(t *testing.T) {
	tests := []struct {
		name   string
		answer inbox.Answer
		want   inbox.Answer
	}{
		{"option", inbox.Answer{Option: "No"}, inbox.Answer{Option: "No"}},
		{
			"text",
			inbox.Answer{Text: "  wait for the review\n"},
			inbox.Answer{Text: "wait for the review"},
		},
		{
			"both",
			inbox.Answer{Option: "Yes", Text: "squash"},
			inbox.Answer{Option: "Yes", Text: "squash"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var box inbox.Inbox

				done := ask(t.Context(), &box, merge())

				if err := box.Answer(only(t, &box).ID, test.answer); err != nil {
					t.Fatalf("Answer() = %v, want nil", err)
				}

				if got := <-done; got.err != nil || got.answer != test.want {
					t.Errorf("Ask() = %+v, %v, want %+v, nil", got.answer, got.err, test.want)
				}

				if pending := box.Pending(); len(pending) != 0 {
					t.Errorf("Pending() = %+v after the answer, want none", pending)
				}
			})
		})
	}
}

func TestPendingListsNewestFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var box inbox.Inbox

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		questions := []inbox.Question{merge(), {Project: "dotfiles", Text: "Which port?"}}
		for _, question := range questions {
			ask(ctx, &box, question)
		}

		pending := box.Pending()
		ids := make([]uuid.UUID, 0, len(pending))

		for i := range pending {
			ids = append(ids, pending[i].ID)
			pending[i].ID = uuid.Nil()
		}

		want := slices.Clone(questions)
		slices.Reverse(want)

		if !reflect.DeepEqual(pending, want) {
			t.Fatalf("Pending() without IDs = %+v, want %+v", pending, want)
		}

		if ids[0] == ids[1] || slices.Contains(ids, uuid.Nil()) {
			t.Errorf("IDs = %v, want two distinct IDs", ids)
		}
	})
}

func TestAskEndsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var box inbox.Inbox

		ctx, cancel := context.WithCancelCause(t.Context())
		done := ask(ctx, &box, merge())
		waiting := only(t, &box)

		cause := errors.New("the agent stopped")
		cancel(cause)

		if got := <-done; !errors.Is(got.err, cause) {
			t.Errorf("Ask() error = %v, want %v", got.err, cause)
		}

		if pending := box.Pending(); len(pending) != 0 {
			t.Errorf("Pending() = %+v after the asker left, want none", pending)
		}

		err := box.Answer(waiting.ID, inbox.Answer{Option: "Yes"})
		if !errors.Is(err, inbox.ErrNotFound) {
			t.Errorf("Answer() after the asker left = %v, want %v", err, inbox.ErrNotFound)
		}
	})
}

func TestAnswerNeedsAWaitingQuestion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var box inbox.Inbox

		err := box.Answer(uuid.New(), inbox.Answer{Option: "Yes"})
		if !errors.Is(err, inbox.ErrNotFound) {
			t.Errorf("Answer() on an empty inbox = %v, want %v", err, inbox.ErrNotFound)
		}

		done := ask(t.Context(), &box, merge())
		waiting := only(t, &box)

		if err := box.Answer(waiting.ID, inbox.Answer{Option: "Yes"}); err != nil {
			t.Fatalf("first Answer() = %v, want nil", err)
		}

		err = box.Answer(waiting.ID, inbox.Answer{Option: "No"})
		if !errors.Is(err, inbox.ErrNotFound) {
			t.Errorf("second Answer() = %v, want %v", err, inbox.ErrNotFound)
		}

		if got := <-done; got.answer.Option != "Yes" {
			t.Errorf("Ask() = %+v, want the first answer", got.answer)
		}
	})
}

func TestEachAskerGetsItsOwnAnswer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var box inbox.Inbox

		const askers = 10

		dones := make([]<-chan result, 0, askers)
		for i := range askers {
			question := inbox.Question{Project: "nudge", Text: fmt.Sprintf("Question %d?", i)}
			dones = append(dones, ask(t.Context(), &box, question))
		}

		var answering sync.WaitGroup
		for _, question := range box.Pending() {
			answering.Go(func() {
				if err := box.Answer(question.ID, inbox.Answer{Text: question.Text}); err != nil {
					t.Errorf("Answer(%q) = %v, want nil", question.Text, err)
				}
			})
		}

		answering.Wait()

		for i, done := range dones {
			want := fmt.Sprintf("Question %d?", i)
			if got := <-done; got.err != nil || got.answer.Text != want {
				t.Errorf("asker %d got %+v, %v, want the text %q", i, got.answer, got.err, want)
			}
		}
	})
}
