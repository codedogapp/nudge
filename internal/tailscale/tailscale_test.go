package tailscale_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/tailscale"
)

// The login and host that the middleware under test lets in.
const (
	login = "user@example.com"
	host  = "machine.tailnet.ts.net"
)

func TestTheLoginOnTheHostPasses(t *testing.T) {
	for _, requestHost := range []string{
		host, host + ":8443", "Machine.Tailnet.TS.NET", "Machine.Tailnet.TS.NET:8443",
	} {
		rec, passed := pass(allow(), request(t, requestHost, login))

		if !passed || rec.Code != http.StatusOK {
			t.Errorf("host %s: got %d, passed %t; want 200, passed", requestHost, rec.Code, passed)
		}
	}
}

func TestEveryOtherRequestIsRefused(t *testing.T) {
	tests := []struct{ name, host, login string }{
		{"no login", host + ":8443", ""},
		{"another login", host + ":8443", "someone@example.com"},
		{"the login in another case", host + ":8443", "User@example.com"},
		{"the loopback host", "127.0.0.1:8700", login},
		{"another host", "other.tailnet.ts.net:8443", login},
		{"a rebound name", "evil.example:8700", login},
		{"the host inside a longer name", host + ".evil.example:8443", login},
		{"the host with two ports", host + ":8443:8443", login},
		{"no host", "", login},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec, passed := pass(allow(), request(t, test.host, test.login))
			assertRefused(t, rec, passed)
		})
	}
}

func TestAnEmptyLoginLetsNobodyIn(t *testing.T) {
	middleware := tailscale.RequireLogin("", host, slog.New(slog.DiscardHandler))

	rec, passed := pass(middleware, request(t, host, ""))
	assertRefused(t, rec, passed)
}

func TestARefusalIsLogged(t *testing.T) {
	var logs strings.Builder

	middleware := tailscale.RequireLogin(login, host, slog.New(slog.NewTextHandler(&logs, nil)))
	pass(middleware, request(t, "evil.example:8700", "someone@example.com"))

	for _, want := range []string{
		"level=WARN", "path=/", "login=someone@example.com", "host=evil.example:8700",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %q, want them to contain %q", logs.String(), want)
		}
	}
}

// allow returns the middleware under test, which lets in login on host.
func allow() func(http.Handler) http.Handler {
	return tailscale.RequireLogin(login, host, slog.New(slog.DiscardHandler))
}

// request returns a GET / with the Host requestHost and, unless requestLogin is empty, a
// Tailscale-User-Login.
func request(t *testing.T, requestHost, requestLogin string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Host = requestHost

	if requestLogin != "" {
		req.Header.Set("Tailscale-User-Login", requestLogin)
	}

	return req
}

// pass sends req through middleware to a handler that answers 200, and reports the response and
// whether req reached that handler.
func pass(
	middleware func(http.Handler) http.Handler,
	req *http.Request,
) (*httptest.ResponseRecorder, bool) {
	passed := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { passed = true })

	rec := httptest.NewRecorder()
	middleware(next).ServeHTTP(rec, req)

	return rec, passed
}

func assertRefused(t *testing.T, rec *httptest.ResponseRecorder, passed bool) {
	t.Helper()

	if passed {
		t.Error("the request reached the next handler, want it refused")
	}

	if rec.Code != http.StatusForbidden || rec.Body.String() != "Forbidden\n" {
		t.Errorf("got %d %q, want 403 %q", rec.Code, rec.Body, "Forbidden\n")
	}

	if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", got)
	}
}
