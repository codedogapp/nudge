package agentdoor

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"

	"github.com/codedogapp/nudge/internal/inbox"
)

// maxQuestion caps a question's body: a few lines of text, and details of a few pages.
const maxQuestion = 64 << 10

// question is a question as nudge ask sends it.
type question struct {
	Project string   `json:"project"`
	Text    string   `json:"text"`
	Details string   `json:"details"`
	Options []string `json:"options"`
}

// answer is an answer as nudge ask receives it.
type answer struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

func routes(box *inbox.Inbox) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /questions", func(w http.ResponseWriter, r *http.Request) {
		ask(w, r, box)
	})

	return mux
}

// ask holds the request until the question has an answer, the client leaves or the server stops.
func ask(w http.ResponseWriter, r *http.Request, box *inbox.Inbox) {
	var asked question

	body := http.MaxBytesReader(w, r.Body, maxQuestion)
	if err := json.UnmarshalRead(body, &asked, json.RejectUnknownMembers(true)); err != nil {
		http.Error(w, "read the question: "+err.Error(), http.StatusBadRequest)
		return
	}

	got, err := box.Ask(r.Context(), inbox.Question{
		Project: asked.Project,
		Text:    asked.Text,
		Details: asked.Details,
		Options: asked.Options,
	})

	switch {
	case err == nil:
		w.Header().Set("Content-Type", "application/json")
		// A reply typed with invalid UTF-8 still gets through, with U+FFFD in its place.
		_ = json.MarshalWrite(w, answer(got), jsontext.AllowInvalidUTF8(true))
	case r.Context().Err() == nil:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		// The server is stopping. If the client has gone instead, no one reads this.
		http.Error(w, "nudge serve is stopping", http.StatusServiceUnavailable)
	}
}
