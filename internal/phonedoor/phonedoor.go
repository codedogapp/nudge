// Package phonedoor is nudge's door for the phone: web pages that show the questions that wait
// for an answer, and take the answers.
package phonedoor

import (
	// embed fills page with page.html, through the go:embed directive below.
	_ "embed"
	"html/template"
	"net/http"
	"time"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/sessions"
	"github.com/codedogapp/nudge/internal/subscriptions"
)

//go:embed page.html
var page string

// site serves the pages from the inbox, and keeps the devices that subscribe to its pushes.
type site struct {
	box     *inbox.Inbox
	live    *sessions.Registry
	pushKey string
	devices *subscriptions.Store
	page    *template.Template
}

// view is what the page shows: why the last answer was refused, if it was, the questions that
// wait and the sessions that run. It also carries the push key.
type view struct {
	PushKey   string
	Refusal   string
	Questions []inbox.Question
	Sessions  []sessionView
}

// Routes returns the phone door's handler. GET / lists the questions that wait, newest first, and
// the sessions in live, each with its last note, and POST /questions/{id}/answer answers one; GET of a file in static gives the page its look, its
// icons and its manifest. The page carries pushKey, the server's VAPID public key, for a browser
// to subscribe to its pushes, and POST /subscriptions adds the subscription to devices. It
// refuses cross-origin posts, and adds the security headers to every response.
func Routes(
	box *inbox.Inbox,
	live *sessions.Registry,
	pushKey string,
	devices *subscriptions.Store,
) http.Handler {
	pages := site{
		box:     box,
		live:    live,
		pushKey: pushKey,
		devices: devices,
		page:    template.Must(template.New("page").Parse(page)),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", pages.list)
	serveStatic(mux)
	mux.HandleFunc("POST /questions/{id}/answer", pages.answer)
	mux.HandleFunc("POST /subscriptions", pages.subscribe)

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
	_ = s.page.Execute(w, view{
		PushKey:   s.pushKey,
		Refusal:   refusal,
		Questions: s.box.Pending(),
		Sessions:  viewSessions(s.live.List(), time.Now()),
	})
}
