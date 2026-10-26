// Package sessions keeps the agent sessions that are running, each with its last note. A session
// lasts as long as its owner keeps it.
package sessions

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"
)

var (
	// ErrNotFound means the session has ended.
	ErrNotFound = errors.New("no such session")
	// ErrNoNote means the note is empty.
	ErrNoNote = errors.New("the note is empty")
)

// Session is a running agent session. Its note is the agent's last report, empty until it makes
// one.
type Session struct {
	ID      uuid.UUID
	Project string
	Note    string
	NotedAt time.Time
}

// Registry holds the running sessions. The zero value is an empty registry, safe for use by
// several goroutines at once.
type Registry struct {
	mu      sync.Mutex
	running []*Session // oldest first
}

// Start adds a session for project, and returns it with the function that ends it. Ending it twice
// is harmless.
func (r *Registry) Start(project string) (Session, func()) {
	started := &Session{ID: uuid.New(), Project: project}

	r.mu.Lock()
	r.running = append(r.running, started)
	r.mu.Unlock()

	return *started, func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.running = slices.DeleteFunc(r.running, func(s *Session) bool { return s == started })
	}
}

// Report sets the session's note, replacing its last one.
func (r *Registry) Report(sessionID uuid.UUID, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return ErrNoNote
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := slices.IndexFunc(r.running, func(s *Session) bool { return s.ID == sessionID })
	if index < 0 {
		return ErrNotFound
	}

	r.running[index].Note = note
	r.running[index].NotedAt = time.Now()

	return nil
}

// List returns the running sessions, newest first.
func (r *Registry) List() []Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	listed := make([]Session, 0, len(r.running))
	for _, s := range slices.Backward(r.running) {
		listed = append(listed, *s)
	}

	return listed
}
