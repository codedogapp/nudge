package agentdoor_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
)

type asked struct {
	answer agentdoor.Answer
	err    error
}

// send asks question through the door's client in a goroutine, and returns once the server has
// done all it can.
func (d *door) send(ctx context.Context, question agentdoor.Question) <-chan asked {
	done := make(chan asked, 1)

	go func() {
		got, err := agentdoor.Ask(ctx, d.client, question)
		done <- asked{got, err}
	}()

	synctest.Wait()

	return done
}

// replying is a transport that gives every request the same reply, without a server.
type replying struct {
	status int
	body   string
}

func (reply replying) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		_ = request.Body.Close()
	}

	return &http.Response{
		StatusCode: reply.status,
		Status:     fmt.Sprintf("%d %s", reply.status, http.StatusText(reply.status)),
		Body:       io.NopCloser(strings.NewReader(reply.body)),
	}, nil
}

func TestAskGetsTheAnswer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		question := agentdoor.Question{
			Project: "nudge",
			Text:    "Merge?",
			Details: "CI is green.",
			Options: []string{"Yes", "No"},
		}
		done := agent.send(t.Context(), question)

		waiting := only(t, &agent.box)
		want := inbox.Question{
			ID:      waiting.ID,
			Project: question.Project,
			Text:    question.Text,
			Details: question.Details,
			Options: question.Options,
		}

		if !reflect.DeepEqual(waiting, want) {
			t.Errorf("the waiting question = %+v, want %+v", waiting, want)
		}

		err := agent.box.Answer(waiting.ID, inbox.Answer{Option: "Yes", Text: "squash"})
		if err != nil {
			t.Fatalf("Answer() = %v, want nil", err)
		}

		answer := agentdoor.Answer{Option: "Yes", Text: "squash"}
		if got := <-done; got.err != nil || got.answer != answer {
			t.Errorf("Ask() = %+v, %v, want %+v", got.answer, got.err, answer)
		}
	})
}

func TestAskEndsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)

		ctx, cancel := context.WithCancel(t.Context())
		done := agent.send(ctx, agentdoor.Question{Text: "Merge?"})
		only(t, &agent.box)

		cancel()

		if got := <-done; !errors.Is(got.err, context.Canceled) {
			t.Errorf("Ask() error = %v, want %v", got.err, context.Canceled)
		}

		synctest.Wait()

		if pending := agent.box.Pending(); len(pending) != 0 {
			t.Errorf("Pending() = %+v after Ask ended, want none", pending)
		}
	})
}

func TestAskGivesTheServersReason(t *testing.T) {
	t.Run("a refused question", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			agent := serve(t.Context(), t)

			got := <-agent.send(t.Context(), agentdoor.Question{Project: "nudge"})
			if want := "the question has no text"; got.err == nil || got.err.Error() != want {
				t.Errorf("Ask() error = %v, want %q", got.err, want)
			}
		})
	})

	t.Run("a server that stops", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, stop := context.WithCancel(t.Context())
			agent := serve(ctx, t)
			done := agent.send(t.Context(), agentdoor.Question{Text: "Merge?"})

			stop()

			got := <-done
			if want := "nudge serve is stopping"; got.err == nil || got.err.Error() != want {
				t.Errorf("Ask() error = %v, want %q", got.err, want)
			}
		})
	})
}

func TestAskReportsWhatFailed(t *testing.T) {
	merge := agentdoor.Question{Text: "Merge?"}
	tests := []struct {
		name     string
		client   *http.Client
		question agentdoor.Question
		want     string
	}{
		{
			"a refusal without a reason",
			&http.Client{Transport: replying{status: http.StatusServiceUnavailable}},
			merge,
			"503 Service Unavailable",
		},
		{
			"an answer that isn't JSON",
			&http.Client{Transport: replying{status: http.StatusOK, body: "Yes"}},
			merge,
			"read the answer: ",
		},
		{
			"a question with invalid UTF-8",
			&http.Client{Transport: replying{status: http.StatusOK, body: "{}"}},
			agentdoor.Question{Text: "\xff"},
			"write the question: ",
		},
		{
			"no server",
			agentdoor.NewClient(socketIn(t)),
			merge,
			"reach nudge serve: ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := agentdoor.Ask(t.Context(), test.client, test.question)
			if err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Errorf("Ask() = %v, want an error that starts with %q", err, test.want)
			}
		})
	}
}

func TestNewClientReachesTheSocket(t *testing.T) {
	path := socketIn(t)
	listener := listen(t, path)
	ctx, stop := context.WithCancel(t.Context())
	served := make(chan error, 1)

	var box inbox.Inbox

	go func() { served <- agentdoor.Serve(ctx, listener, &box) }()

	// A question without text gets its refusal at once, so no one has to answer it.
	_, err := agentdoor.Ask(t.Context(), agentdoor.NewClient(path), agentdoor.Question{})
	if want := "the question has no text"; err == nil || err.Error() != want {
		t.Errorf("Ask() error = %v, want %q", err, want)
	}

	stop()

	if err := <-served; err != nil {
		t.Errorf("Serve() = %v, want nil", err)
	}
}
