package agentdoor_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

func TestSessionLastsAsLongAsItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		ctx, end := context.WithCancel(t.Context())

		session, err := agentdoor.Register(ctx, agent.client, "nudge")
		if err != nil {
			t.Fatalf("Register() = %v, want nil", err)
		}

		if err := session.Report(t.Context(), "running the tests"); err != nil {
			t.Fatalf("Report() = %v, want nil", err)
		}

		listed := agent.live.List()
		if len(listed) != 1 || listed[0].Project != "nudge" ||
			listed[0].Note != "running the tests" {
			t.Fatalf("List() = %+v, want the nudge session with its note", listed)
		}

		end()

		// The server notices the closed connection once the bubble's clock moves on.
		time.Sleep(time.Second)
		synctest.Wait()

		if listed := agent.live.List(); len(listed) != 0 {
			t.Errorf("List() after the context ended = %+v, want none", listed)
		}
	})
}

func TestSessionRefusals(t *testing.T) {
	missing := uuid.New().String()
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"a start that isn't JSON", http.MethodPost, "/sessions", "{", http.StatusBadRequest},
		{
			"an ID that isn't one",
			http.MethodPut,
			"/sessions/x/note",
			`{"note":"a"}`,
			http.StatusNotFound,
		},
		{
			"a session that isn't there",
			http.MethodPut,
			"/sessions/" + missing + "/note",
			`{"note":"a"}`,
			http.StatusNotFound,
		},
		{
			"a report that isn't JSON",
			http.MethodPut,
			"/sessions/" + missing + "/note",
			"{",
			http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				agent := serve(t.Context(), t)

				got := agent.do(t.Context(), test.method, test.path, test.body)
				if got.err != nil || got.status != test.want {
					t.Errorf("reply = %d, %v, want %d", got.status, got.err, test.want)
				}
			})
		})
	}
}

func TestReportWithoutANoteIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)

		session, err := agentdoor.Register(t.Context(), agent.client, "nudge")
		if err != nil {
			t.Fatalf("Register() = %v, want nil", err)
		}

		err = session.Report(t.Context(), " ")
		if want := "the note is empty"; err == nil || err.Error() != want {
			t.Errorf("Report() = %v, want %q", err, want)
		}
	})
}

func TestSessionClientReportsTheServersRefusal(t *testing.T) {
	client := &http.Client{
		Transport: replying{http.StatusServiceUnavailable, "nudge serve is stopping\n"},
	}

	_, err := agentdoor.Register(t.Context(), client, "nudge")
	if err == nil || !strings.HasPrefix(err.Error(), "nudge serve is stopping") {
		t.Errorf("Register() = %v, want the server's reason", err)
	}
}
