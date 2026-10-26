package cli

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/agentdoor"
	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/sessions"
	"github.com/codedogapp/nudge/internal/tailscale"
)

const (
	testLogin = "user@example.com"
	testHost  = "machine.tailnet.ts.net"
)

func tailnetServer(t *testing.T, box *inbox.Inbox) *httptest.Server {
	t.Helper()

	phone := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the page"))
	})
	lock := tailscale.RequireLogin(testLogin, testHost, slog.New(slog.DiscardHandler))
	server := httptest.NewServer(
		lock(tailnetDoor(phone, agentdoor.Routes(box, &sessions.Registry{}))),
	)
	t.Cleanup(server.Close)

	return server
}

func post(t *testing.T, url, login, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		url,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}

	req.Host = testHost
	if login != "" {
		req.Header.Set("Tailscale-User-Login", login)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func TestAnAgentOnTheTailnetAsksAndGetsTheAnswer(t *testing.T) {
	box := &inbox.Inbox{}
	server := tailnetServer(t, box)
	done := make(chan *http.Response, 1)

	go func() { done <- post(t, server.URL+"/agent/questions", testLogin, `{"text":"Merge?"}`) }()

	var pending []inbox.Question
	for len(pending) == 0 {
		runtime.Gosched()

		pending = box.Pending()
	}

	if err := box.Answer(pending[0].ID, inbox.Answer{Text: "yes"}); err != nil {
		t.Fatal(err)
	}

	if resp := <-done; resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestTheTailnetDoorKeepsItsLock(t *testing.T) {
	server := tailnetServer(t, &inbox.Inbox{})

	if resp := post(
		t,
		server.URL+"/agent/questions",
		"",
		`{"text":"Merge?"}`,
	); resp.StatusCode != http.StatusForbidden {
		t.Errorf("no login: status = %d, want 403", resp.StatusCode)
	}

	if resp := post(
		t,
		server.URL+"/agent/questions",
		"other@example.com",
		`{}`,
	); resp.StatusCode != http.StatusForbidden {
		t.Errorf("another login: status = %d, want 403", resp.StatusCode)
	}
}

func TestThePhoneDoorStaysAtTheRoot(t *testing.T) {
	server := tailnetServer(t, &inbox.Inbox{})

	if resp := post(t, server.URL+"/", testLogin, ""); resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
