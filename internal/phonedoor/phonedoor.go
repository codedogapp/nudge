// Package phonedoor is nudge's door for the phone: web pages that show the questions that wait
// for an answer, and take the answers.
package phonedoor

import (
	// embed fills page with page.html, through the go:embed directive below.
	_ "embed"
	"html/template"
	"net/http"

	"github.com/codedogapp/nudge/internal/inbox"
)

//go:embed page.html
var page string

// site serves the pages from the inbox.
type site struct {
	box  *inbox.Inbox
	page *template.Template
}

// view is what the page shows: why the last answer was refused, if it was, and the questions that
// wait.
type view struct {
	Refusal   string
	Questions []inbox.Question
}

// Routes returns the phone door's handler. GET / lists the questions that wait, newest first, and
// POST /questions/{id}/answer answers one. It refuses cross-origin posts, and adds the security
// headers to every response.
func Routes(box *inbox.Inbox) http.Handler {
	pages := site{box: box, page: template.Must(template.New("page").Parse(page))}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", pages.list)
	mux.HandleFunc("POST /questions/{id}/answer", pages.answer)

	return secure(http.NewCrossOriginProtection().Handler(mux))
}

func (s site) list(w http.ResponseWriter, _ *http.Request) {
	s.show(w, http.StatusOK, "")
}

// show writes the page with status: the refusal, if there is one, and the questions that wait.
func (s site) show(w http.ResponseWriter, status int, refusal string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page shows live questions and their details: no cache keeps a copy.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	// Only a client that has gone can make it fail, and then no one reads the page.
	_ = s.page.Execute(w, view{Refusal: refusal, Questions: s.box.Pending()})
}
