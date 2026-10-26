package agentdoor

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"uuid"

	"github.com/codedogapp/nudge/internal/sessions"
)

// startSession starts a session and tells the client its ID. It then holds the request open: the
// session lasts until the client leaves or the server stops.
func startSession(w http.ResponseWriter, r *http.Request, live *sessions.Registry) {
	var start Start

	body := http.MaxBytesReader(w, r.Body, maxQuestion)
	if err := json.UnmarshalRead(body, &start, json.RejectUnknownMembers(true)); err != nil {
		http.Error(w, "read the start: "+err.Error(), http.StatusBadRequest)
		return
	}

	session, end := live.Start(start.Project)
	defer end()

	w.Header().Set("Content-Type", "application/json")

	if err := json.MarshalWrite(w, Started{ID: session.ID.String()}); err != nil {
		return
	}

	if err := http.NewResponseController(w).Flush(); err != nil {
		return
	}

	<-r.Context().Done()
}

// report sets the note of the session in the path.
func report(w http.ResponseWriter, r *http.Request, live *sessions.Registry) {
	sessionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, sessions.ErrNotFound.Error(), http.StatusNotFound)
		return
	}

	var noted Report

	body := http.MaxBytesReader(w, r.Body, maxQuestion)
	if err := json.UnmarshalRead(body, &noted, json.RejectUnknownMembers(true)); err != nil {
		http.Error(w, "read the report: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch err := live.Report(sessionID, noted.Note); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, sessions.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
