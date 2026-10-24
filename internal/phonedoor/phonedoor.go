// Package phonedoor is nudge's door for the phone: web pages that show the questions that wait
// for an answer.
package phonedoor

import (
	_ "embed"
	"html/template"
	"net/http"

	"github.com/codedogapp/nudge/internal/inbox"
)

//go:embed page.html
var page string

// Routes returns the phone door's handler. GET / lists the questions that wait, newest first.
func Routes(box *inbox.Inbox) http.Handler {
	list := template.Must(template.New("page").Parse(page))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Only a client that has gone can make it fail, and then no one reads the page.
		_ = list.Execute(w, box.Pending())
	})

	return mux
}
