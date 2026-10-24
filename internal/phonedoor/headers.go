package phonedoor

import "net/http"

// policy is the pages' Content-Security-Policy: they load only their scripts, their stylesheet, its
// grain, their icons and their manifest, connect and post only to themselves, and no other page can
// frame them. The service worker falls under script-src. base-uri, form-action and frame-ancestors
// don't fall back to default-src, so the policy names them.
const policy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"manifest-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; " +
	"frame-ancestors 'none'"

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
