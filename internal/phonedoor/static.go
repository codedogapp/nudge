package phonedoor

import (
	"embed"
	"net/http"
)

// static holds the files that the door serves as they are: the page's look, its icons and its
// manifest.
//
//go:embed static
var static embed.FS

// serveStatic routes GET /<name> to each file in static. The list is explicit, so the door serves
// nothing else and lists no folder.
func serveStatic(mux *http.ServeMux) {
	for _, name := range []string{
		"style.css", "grain.svg", "icon.svg", "apple-touch-icon.png", "icon-192.png",
		"icon-512.png", "manifest.json",
	} {
		mux.HandleFunc("GET /"+name, asset(name))
	}
}

// asset serves the embedded file name. A new binary can bring a new look, so browsers check it
// each time.
func asset(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, static, "static/"+name)
	}
}
