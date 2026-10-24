package agentdoor

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// questionsURL is where a client posts its questions. The client dials the socket whatever the
// host says.
const questionsURL = "http://nudge/questions"

// NewClient returns a client that reaches the agent door on the socket at path. It has no time
// limit: a question waits for as long as the user takes to answer it.
func NewClient(path string) *http.Client {
	dialer := &net.Dialer{}

	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", path)
		},
	}}
}

// Ask asks question through the agent door, and waits for its answer. When ctx ends, the request
// ends, and the question leaves the inbox. A refusal becomes an error with the server's reason.
func Ask(ctx context.Context, client *http.Client, question Question) (Answer, error) {
	body, err := json.Marshal(question)
	if err != nil {
		return Answer{}, fmt.Errorf("write the question: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, questionsURL, bytes.NewReader(body))
	if err != nil {
		return Answer{}, fmt.Errorf("make the request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return Answer{}, fmt.Errorf("reach nudge serve: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		reason, _ := io.ReadAll(response.Body)
		return Answer{}, errors.New(cmp.Or(strings.TrimSpace(string(reason)), response.Status))
	}

	var got Answer
	if err := json.UnmarshalRead(response.Body, &got); err != nil {
		return Answer{}, fmt.Errorf("read the answer: %w", err)
	}

	return got, nil
}
