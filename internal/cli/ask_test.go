package cli_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/cli"
)

// fakeDoor serves a fake agent door at NUDGE_SOCKET. It replies to every question with status and
// body, and hands the question over.
func fakeDoor(t *testing.T, status int, body string) <-chan agentdoor.Question {
	t.Helper()

	listener, err := agentdoor.Listen(t.Context(), socketIn(t))
	if err != nil {
		t.Fatal(err)
	}

	asked := make(chan agentdoor.Question, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var question agentdoor.Question
			if err := json.UnmarshalRead(r.Body, &question); err != nil {
				t.Errorf("read the question: %v", err)
			}

			asked <- question

			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		}))

	_ = server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)

	return asked
}

// brokenPipe is a stdout that no one reads anymore.
type brokenPipe struct{}

func (brokenPipe) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestAskSendsTheQuestion(t *testing.T) {
	const private os.FileMode = 0o700

	asked := fakeDoor(t, http.StatusOK, `{"option":"Yes","text":""}`)

	project := filepath.Join(t.TempDir(), "dotfiles")
	if err := os.Mkdir(project, private); err != nil {
		t.Fatal(err)
	}

	t.Chdir(project)

	var stdout, stderr strings.Builder

	args := []string{
		"ask",
		"--option",
		"Yes",
		"--option",
		"No",
		"--details",
		"CI is green.",
		"Merge?",
	}
	if code := cli.Run(t.Context(), args, &stdout, &stderr); code != 0 {
		t.Errorf("Run(ask) = %d, stderr = %q, want 0", code, stderr.String())
	}

	want := agentdoor.Question{
		Project: "dotfiles",
		Text:    "Merge?",
		Details: "CI is green.",
		Options: []string{"Yes", "No"},
	}
	if got := <-asked; !reflect.DeepEqual(got, want) {
		t.Errorf("the question sent = %+v, want %+v", got, want)
	}
}

func TestAskPrintsTheAnswer(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  string
	}{
		{"an option and a text", `{"option":"Yes","text":"squash"}`, "option: Yes\ntext: squash\n"},
		{"an option", `{"option":"No","text":""}`, "option: No\n"},
		{"a text", `{"option":"","text":"later"}`, "text: later\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeDoor(t, http.StatusOK, test.reply)

			var stdout, stderr strings.Builder

			code := cli.Run(t.Context(), []string{"ask", "Merge?"}, &stdout, &stderr)
			if code != 0 || stdout.String() != test.want || stderr.Len() != 0 {
				t.Errorf("Run(ask) = %d, stdout = %q, stderr = %q, want 0 and %q",
					code, stdout.String(), stderr.String(), test.want)
			}
		})
	}
}

func TestAskReportsARefusal(t *testing.T) {
	tests := []struct {
		name   string
		status int
		reply  string
		want   string
	}{
		{
			"a refused question",
			http.StatusBadRequest,
			"the question has no text\n",
			"nudge ask: the question has no text\n",
		},
		{
			"a server that stops",
			http.StatusServiceUnavailable,
			"nudge serve is stopping\n",
			"nudge ask: nudge serve is stopping\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeDoor(t, test.status, test.reply)

			var stdout, stderr strings.Builder

			code := cli.Run(t.Context(), []string{"ask", "Merge?"}, &stdout, &stderr)
			if code != 1 || stderr.String() != test.want || stdout.Len() != 0 {
				t.Errorf("Run(ask) = %d, stdout = %q, stderr = %q, want 1 and %q",
					code, stdout.String(), stderr.String(), test.want)
			}
		})
	}
}

func TestAskNeedsTheServer(t *testing.T) {
	socketIn(t)

	var stdout, stderr strings.Builder

	code := cli.Run(t.Context(), []string{"ask", "Merge?"}, &stdout, &stderr)
	if want := "nudge ask: reach nudge serve: "; code != 1 ||
		!strings.HasPrefix(stderr.String(), want) {
		t.Errorf("Run(ask) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}
}

func TestAskReportsALostAnswer(t *testing.T) {
	fakeDoor(t, http.StatusOK, `{"option":"Yes","text":""}`)

	var stderr strings.Builder

	code := cli.Run(t.Context(), []string{"ask", "Merge?"}, brokenPipe{}, &stderr)
	if want := "nudge ask: print the answer: " + io.ErrClosedPipe.Error() + "\n"; code != 1 ||
		stderr.String() != want {
		t.Errorf("Run(ask) = %d, stderr = %q, want 1 and %q", code, stderr.String(), want)
	}
}
