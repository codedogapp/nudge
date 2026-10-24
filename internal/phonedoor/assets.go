package phonedoor

import (
	"embed"
	"net/http"
)

// assets holds the files that the page loads for its look.
//
//go:embed style.css grain.svg
var assets embed.FS

// asset serves the embedded file name. A new binary can bring a new look, so browsers check it
// each time.
func asset(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, assets, name)
	}
}
