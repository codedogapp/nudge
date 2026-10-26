package phonedoor

import (
	"fmt"
	"time"

	"github.com/codedogapp/nudge/internal/sessions"
)

// sessionView is a running session as the page shows it: its project, its last note, and how long
// ago it made it. A session that hasn't reported has neither.
type sessionView struct {
	Project string
	Note    string
	Age     string
}

// viewSessions turns the running sessions into what the page shows, as of now.
func viewSessions(running []sessions.Session, now time.Time) []sessionView {
	views := make([]sessionView, 0, len(running))

	for _, session := range running {
		view := sessionView{Project: session.Project, Note: session.Note}
		if session.Note != "" {
			view.Age = age(now.Sub(session.NotedAt))
		}

		views = append(views, view)
	}

	return views
}

const day = 24 * time.Hour

// age says how long ago something happened, in its largest whole unit.
func age(elapsed time.Duration) string {
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%d min ago", int(elapsed/time.Minute))
	case elapsed < day:
		return fmt.Sprintf("%d h ago", int(elapsed/time.Hour))
	default:
		return fmt.Sprintf("%d d ago", int(elapsed/day))
	}
}
