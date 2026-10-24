package mcptool_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/nudge/internal/agentdoor"
)

// call is a question that waits in a fake agent door for its answer.
type call struct {
	question agentdoor.Question
	answer   chan agentdoor.Answer
}

// door is a fake agent door: it hands each question over, and returns the answer sent to it, or
// the error of its context when that ends first. ended says each time a context has ended.
type door struct {
	calls chan call
	ended chan struct{}
}

func newDoor() door {
	return door{calls: make(chan call), ended: make(chan struct{}, 2)}
}

func (d door) ask(ctx context.Context, question agentdoor.Question) (agentdoor.Answer, error) {
	waiting := call{question: question, answer: make(chan agentdoor.Answer)}
	d.calls <- waiting

	select {
	case answer := <-waiting.answer:
		return answer, nil
	case <-ctx.Done():
		d.ended <- struct{}{}
		return agentdoor.Answer{}, ctx.Err()
	}
}

// outcome is how a call of the ask tool ended.
type outcome struct {
	result *mcp.CallToolResult
	err    error
}

// startCall calls the ask tool with question in the background, with ctx, and returns where its
// outcome arrives.
func startCall(ctx context.Context, session *mcp.ClientSession, question string) <-chan outcome {
	ended := make(chan outcome, 1)

	go func() {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "ask", Arguments: map[string]any{"question": question},
		})
		ended <- outcome{result: result, err: err}
	}()

	return ended
}

// answerOf returns the text of a call's answer, and fails the test when the call failed.
func answerOf(t *testing.T, ended <-chan outcome) string {
	t.Helper()

	got := <-ended
	if got.err != nil {
		t.Fatal(got.err)
	}

	return textOf(t, got.result)
}

func TestCancelEndsTheQuestion(t *testing.T) {
	fake := newDoor()
	ctx, cancel := context.WithCancel(t.Context())

	ended := startCall(ctx, connect(t, fake.ask), "Ship it?")

	<-fake.calls
	cancel()
	<-fake.ended

	if got := <-ended; !errors.Is(got.err, context.Canceled) {
		t.Errorf("call = %v, want it cancelled", got.err)
	}
}

func TestAnAgentThatGoesAwayEndsTheQuestion(t *testing.T) {
	fake := newDoor()
	serverEnd, agentEnd := net.Pipe()
	ran := serve(t.Context(), fake.ask, &mcp.IOTransport{Reader: serverEnd, Writer: serverEnd})

	ended := startCall(t.Context(), join(t, &mcp.IOTransport{Reader: agentEnd, Writer: agentEnd}),
		"Ship it?")

	<-fake.calls

	if err := agentEnd.Close(); err != nil {
		t.Fatal(err)
	}

	<-fake.ended

	if err := <-ran; err != nil {
		t.Errorf("Run = %v, want nil once the agent has gone", err)
	}

	if got := <-ended; got.err == nil {
		t.Error("call = nil, want an error once the agent's end has closed")
	}
}

func TestStoppingTheServerEndsTheQuestion(t *testing.T) {
	fake := newDoor()
	ctx, stop := context.WithCancel(t.Context())
	serverEnd, clientEnd := mcp.NewInMemoryTransports()
	ran := serve(ctx, fake.ask, serverEnd)

	ended := startCall(t.Context(), join(t, clientEnd), "Ship it?")

	<-fake.calls
	stop()
	<-fake.ended

	if err := <-ran; !errors.Is(err, context.Canceled) {
		t.Errorf("Run = %v, want it cancelled", err)
	}

	if got := <-ended; got.err == nil && !got.result.IsError {
		t.Errorf("call = %q, want an error once the server has stopped", textOf(t, got.result))
	}
}

func TestTwoCallsAreTwoQuestions(t *testing.T) {
	fake := newDoor()
	session := connect(t, fake.ask)

	first := startCall(t.Context(), session, "First?")
	second := startCall(t.Context(), session, "Second?")

	waiting := map[string]call{}

	for range 2 {
		asked := <-fake.calls
		waiting[asked.question.Text] = asked
	}

	waiting["Second?"].answer <- agentdoor.Answer{Text: "two"}

	if text := answerOf(t, second); text != `{"option":"","text":"two"}` {
		t.Errorf("second = %s, want its own answer", text)
	}

	waiting["First?"].answer <- agentdoor.Answer{Text: "one"}

	if text := answerOf(t, first); text != `{"option":"","text":"one"}` {
		t.Errorf("first = %s, want its own answer", text)
	}
}
