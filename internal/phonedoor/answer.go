package phonedoor

import (
	"errors"
	"net/http"
	"strings"
	"uuid"

	"github.com/codedogapp/nudge/internal/inbox"
)

// maxAnswer caps an answer's form: a reply of a few pages.
const maxAnswer = 64 << 10

// gone is the page's reason for refusing an answer to a question that has left the inbox.
const gone = "This question has gone: it was answered, or its agent stopped waiting."

// answer hands the form's answer to the question's asker, and sends the browser back to the list.
// A refused answer gets the list again, with the reason.
func (s site) answer(w http.ResponseWriter, r *http.Request) {
	questionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.show(w, http.StatusNotFound, gone)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAnswer)
	if err := r.ParseForm(); err != nil {
		s.refuse(w, err)
		return
	}

	sent := inbox.Answer{Option: field(r, "option"), Text: field(r, "text")}
	if err := s.box.Answer(questionID, sent); err != nil {
		s.refuse(w, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// field is the form's value for name. Browsers send its line ends as CRLF, so they go back to LF,
// as the agent wrote the options and reads the reply.
func field(r *http.Request, name string) string {
	return strings.ReplaceAll(r.PostForm.Get(name), "\r\n", "\n")
}

// refuse shows the list again, with the reason why the answer was refused.
func (s site) refuse(w http.ResponseWriter, err error) {
	if errors.Is(err, inbox.ErrNotFound) {
		s.show(w, http.StatusNotFound, gone)
		return
	}

	s.show(w, http.StatusBadRequest, "The answer was refused: "+err.Error()+".")
}
