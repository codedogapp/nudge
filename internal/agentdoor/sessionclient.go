package agentdoor

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const sessionsURL = "http://nudge/sessions"

// Session is a session that an agent has started on the agent door. It lasts until the context
// that started it ends.
type Session struct {
	id     string
	client *http.Client
}

// Register starts a session for project through the agent door. The session lasts until ctx ends
// or nudge serve stops.
func Register(ctx context.Context, client *http.Client, project string) (Session, error) {
	body, err := json.Marshal(Start{Project: project})
	if err != nil {
		return Session{}, fmt.Errorf("write the start: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, sessionsURL, bytes.NewReader(body))
	if err != nil {
		return Session{}, fmt.Errorf("make the request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return Session{}, fmt.Errorf("reach nudge serve: %w", err)
	}

	// The body stays open: it keeps the session alive.
	stop := context.AfterFunc(ctx, func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		stop()
		return Session{}, refusal(response)
	}

	// Decode one value only: the body goes on, open, for as long as the session lasts.
	var started Started
	if err := json.UnmarshalDecode(jsontext.NewDecoder(response.Body), &started); err != nil {
		_ = response.Body.Close()
		return Session{}, fmt.Errorf("read the session: %w", err)
	}

	return Session{id: started.ID, client: client}, nil
}

// Report sets the session's note.
func (s Session) Report(ctx context.Context, note string) error {
	body, err := json.Marshal(Report{Note: note})
	if err != nil {
		return fmt.Errorf("write the report: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPut, sessionsURL+"/"+s.id+"/note", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("make the request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("reach nudge serve: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNoContent {
		return refusal(response)
	}

	return nil
}

// refusal turns a reply that isn't a success into an error with the server's reason.
func refusal(response *http.Response) error {
	reason, _ := io.ReadAll(response.Body)

	return errors.New(cmp.Or(strings.TrimSpace(string(reason)), response.Status))
}
