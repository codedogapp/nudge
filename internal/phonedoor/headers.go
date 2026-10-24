package phonedoor

import "net/http"

// policy is the pages' Content-Security-Policy: they load nothing, post only to themselves, and
// no other page can frame them. base-uri, form-action and frame-ancestors don't fall back to
// default-src, so the policy names them.
const policy = "default-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// secure adds the security headers to every response that next sends.
func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", policy)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
