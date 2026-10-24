package inbox_test

import (
	"errors"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestAskRefusesInvalidQuestions(t *testing.T) {
	tests := []struct {
		name     string
		question inbox.Question
		want     error
	}{
		{"no text", inbox.Question{Project: "nudge"}, inbox.ErrNoText},
		{"blank text", inbox.Question{Text: " \n\t"}, inbox.ErrNoText},
		{
			"empty option",
			inbox.Question{Text: "Merge?", Options: []string{"Yes", ""}},
			inbox.ErrBadOptions,
		},
		{
			"blank option",
			inbox.Question{Text: "Merge?", Options: []string{" "}},
			inbox.ErrBadOptions,
		},
		{
			"repeated option",
			inbox.Question{Text: "Merge?", Options: []string{"Yes", "No", "Yes"}},
			inbox.ErrBadOptions,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// In a bubble, an Ask that waits instead of refusing fails the test as a deadlock.
			synctest.Test(t, func(t *testing.T) {
				var box inbox.Inbox

				if _, err := box.Ask(t.Context(), test.question); !errors.Is(err, test.want) {
					t.Errorf("Ask() = %v, want %v", err, test.want)
				}

				if pending := box.Pending(); len(pending) != 0 {
					t.Errorf("Pending() = %+v, want none", pending)
				}
			})
		})
	}
}

func TestAnswerRefusals(t *testing.T) {
	yesNo := []string{"Yes", "No"}
	tests := []struct {
		name    string
		options []string
		answer  inbox.Answer
		want    error
	}{
		{"empty", yesNo, inbox.Answer{}, inbox.ErrEmptyAnswer},
		{"blank text", yesNo, inbox.Answer{Text: " \n\t"}, inbox.ErrEmptyAnswer},
		{"unknown option", yesNo, inbox.Answer{Option: "Maybe"}, inbox.ErrUnknownOption},
		{"option in another case", yesNo, inbox.Answer{Option: "yes"}, inbox.ErrUnknownOption},
		{
			"option without options",
			nil,
			inbox.Answer{Option: "Yes", Text: "ok"},
			inbox.ErrUnknownOption,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var box inbox.Inbox

				question := merge()
				question.Options = test.options
				done := ask(t.Context(), &box, question)
				waiting := only(t, &box)

				if err := box.Answer(waiting.ID, test.answer); !errors.Is(err, test.want) {
					t.Errorf("Answer(%+v) = %v, want %v", test.answer, err, test.want)
				}

				if err := box.Answer(waiting.ID, inbox.Answer{Text: "later"}); err != nil {
					t.Fatalf("Answer() after a refusal = %v, want the question still waiting", err)
				}

				if got := <-done; got.answer.Text != "later" {
					t.Errorf("Ask() = %+v, want the answer after the refusal", got.answer)
				}
			})
		})
	}
}
