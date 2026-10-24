// Package tailscale lets through only the requests that tailscale serve sends for the one login
// allowed in: the phone door's lock (docs/plan.md §3, decision 8).
package tailscale

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// loginHeader carries the requester's Tailscale login. tailscale serve sets it on every request
// from the tailnet and drops any copy the client sent; requests through Funnel or from a tagged
// device arrive without it.
const loginHeader = "Tailscale-User-Login"

// RequireLogin returns middleware that passes a request on only when its Tailscale-User-Login is
// login and its Host, without a port, is host: the ts.net name that tailscale serve publishes.
// A request without a login never passes. Any other request gets a 403 and goes no further, and
// logger gets the login and host it came with.
func RequireLogin(login, host string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get(loginHeader)
			if got == "" || got != login || !strings.EqualFold(hostname(r.Host), host) {
				logger.WarnContext(r.Context(), "refused a request",
					"path", r.URL.Path, "login", got, "host", r.Host)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hostname returns hostport without its port. tailscale serve passes on the browser's Host,
// which carries the port when it isn't 443.
func hostname(hostport string) string {
	name, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}

	return name
}
