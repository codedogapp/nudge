package phonedoor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestEveryResponseCarriesTheSecurityHeaders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box, _ := asked(t, inbox.Question{Text: "Merge?", Options: []string{"Squash"}})
		path := answerPath(box)

		visit := func(path string) *http.Request {
			return httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		}

		crossSite := form(t, path, url.Values{"option": {"Squash"}})
		crossSite.Header.Set("Sec-Fetch-Site", "cross-site")

		tests := []struct {
			name   string
			req    *http.Request
			status int
		}{
			{"the page", visit("/"), http.StatusOK},
			{"the stylesheet", visit("/style.css"), http.StatusOK},
			{"the grain", visit("/grain.svg"), http.StatusOK},
			{"the icon", visit("/icon-192.png"), http.StatusOK},
			{"the manifest", visit("/manifest.json"), http.StatusOK},
			{"a refusal", form(t, path, url.Values{"option": {"Merge"}}), http.StatusBadRequest},
			{"a cross-origin post", crossSite, http.StatusForbidden},
			{"a subscription", subscribing(t, browser), http.StatusNoContent},
			{"no page", visit("/nope"), http.StatusNotFound},
			// Last: it takes the question out of the inbox.
			{"an answer", form(t, path, url.Values{"option": {"Squash"}}), http.StatusSeeOther},
		}

		want := map[string]string{
			"Content-Security-Policy": "default-src 'none'; style-src 'self'; img-src 'self'; " +
				"manifest-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "same-origin",
		}

		for _, test := range tests {
			rec := serve(t, box, test.req)
			if rec.Code != test.status {
				t.Errorf("%s = %d, want %d", test.name, rec.Code, test.status)
			}

			for name, value := range want {
				if got := rec.Header().Get(name); got != value {
					t.Errorf("%s: %s = %q, want %q", test.name, name, got, value)
				}
			}
		}
	})
}
