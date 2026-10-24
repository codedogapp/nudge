package inbox_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestNotifySeesEachNewQuestionOnceItsPending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var box inbox.Inbox

		notified := make(chan inbox.Question, 2)
		box.Notify = func(question inbox.Question) {
			isIt := func(pending inbox.Question) bool { return pending.ID == question.ID }
			if !slices.ContainsFunc(box.Pending(), isIt) {
				t.Errorf("Notify(%+v) came before the question was pending", question)
			}

			notified <- question
		}

		ctx, cancel := context.WithCancel(t.Context())
		done := ask(ctx, &box, merge())

		if got, want := <-notified, only(t, &box); !reflect.DeepEqual(got, want) {
			t.Errorf("Notify(%+v), want Notify(%+v), with its ID", got, want)
		}

		if _, err := box.Ask(ctx, inbox.Question{Text: " "}); !errors.Is(err, inbox.ErrNoText) {
			t.Errorf("Ask(blank) = %v, want %v", err, inbox.ErrNoText)
		}

		cancel()
		<-done

		if len(notified) != 0 {
			t.Errorf("Notify(%+v) for a refused question", <-notified)
		}
	})
}
