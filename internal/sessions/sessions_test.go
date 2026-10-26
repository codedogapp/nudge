package sessions_test

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/codedogapp/nudge/internal/sessions"
)

func projects(list []sessions.Session) []string {
	names := make([]string, 0, len(list))
	for _, s := range list {
		names = append(names, s.Project)
	}

	return names
}

func TestListShowsRunningSessionsNewestFirst(t *testing.T) {
	var registry sessions.Registry

	_, endFirst := registry.Start("first")
	registry.Start("second")

	if got := projects(registry.List()); len(got) != 2 || got[0] != "second" || got[1] != "first" {
		t.Errorf("projects = %v, want [second first]", got)
	}

	endFirst()
	endFirst()

	if got := projects(registry.List()); len(got) != 1 || got[0] != "second" {
		t.Errorf("projects after the first ended = %v, want [second]", got)
	}
}

func TestReportReplacesTheNote(t *testing.T) {
	var registry sessions.Registry

	started, _ := registry.Start("nudge")
	before := time.Now()

	for _, note := range []string{"  reading the code \n", "running the tests"} {
		if err := registry.Report(started.ID, note); err != nil {
			t.Fatalf("Report(%q) = %v, want nil", note, err)
		}
	}

	got := registry.List()[0]
	if got.Note != "running the tests" {
		t.Errorf("Note = %q, want the last note", got.Note)
	}

	if got.NotedAt.Before(before) {
		t.Errorf("NotedAt = %v, want a time after %v", got.NotedAt, before)
	}
}

func TestReportFailures(t *testing.T) {
	var registry sessions.Registry

	started, end := registry.Start("nudge")
	end()

	tests := []struct {
		name string
		id   uuid.UUID
		note string
		want error
	}{
		{"an ended session", started.ID, "hello", sessions.ErrNotFound},
		{"an unknown session", uuid.New(), "hello", sessions.ErrNotFound},
		{"an empty note", started.ID, " \n", sessions.ErrNoNote},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := registry.Report(test.id, test.note); !errors.Is(err, test.want) {
				t.Errorf("Report() = %v, want %v", err, test.want)
			}
		})
	}
}
