package agentdoor_test

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

// forgetting is a transport that answers the first report with 404, as a server that has
// restarted does, and passes everything else on.
type forgetting struct {
	inner     http.RoundTripper
	forgotten atomic.Bool
}

func (f *forgetting) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodPut || !f.forgotten.CompareAndSwap(false, true) {
		return f.inner.RoundTrip(request)
	}

	_ = request.Body.Close()

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       io.NopCloser(strings.NewReader("no such session\n")),
	}, nil
}

func TestReporterStartsASessionOnItsFirstReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		reporter := agentdoor.NewReporter(t.Context(), agent.client, "nudge")

		if err := reporter.Report(t.Context(), "reading the code"); err != nil {
			t.Fatalf("Report() = %v, want nil", err)
		}

		if listed := agent.live.List(); len(listed) != 1 || listed[0].Note != "reading the code" {
			t.Errorf("List() = %+v, want one session with the note", listed)
		}
	})
}

func TestReporterStartsANewSessionWhenTheServerForgotIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		client := &http.Client{Transport: &forgetting{inner: agent.client.Transport}}
		reporter := agentdoor.NewReporter(t.Context(), client, "nudge")

		if err := reporter.Register(); err != nil {
			t.Fatalf("Register() = %v, want nil", err)
		}

		if err := reporter.Report(t.Context(), "running the tests"); err != nil {
			t.Fatalf("Report() = %v, want nil", err)
		}

		// The server drops the old session once the bubble's clock moves on.
		time.Sleep(time.Second)
		synctest.Wait()

		listed := agent.live.List()
		if len(listed) != 1 || listed[0].Note != "running the tests" {
			t.Errorf("List() = %+v, want the new session alone, with the note", listed)
		}
	})
}

func TestReporterFailsWhenTheServerIsAway(t *testing.T) {
	client := &http.Client{
		Transport: replying{http.StatusServiceUnavailable, "nudge serve is stopping\n"},
	}
	reporter := agentdoor.NewReporter(t.Context(), client, "nudge")

	if err := reporter.Report(t.Context(), "hello"); err == nil {
		t.Error("Report() = nil, want an error")
	}
}

func TestReporterKeepsOtherRefusals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent := serve(t.Context(), t)
		reporter := agentdoor.NewReporter(t.Context(), agent.client, "nudge")

		err := reporter.Report(t.Context(), " ")
		if want := "the note is empty"; err == nil || err.Error() != want {
			t.Errorf("Report() = %v, want %q", err, want)
		}
	})
}
