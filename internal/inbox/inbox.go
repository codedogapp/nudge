// Package inbox keeps the questions that wait for an answer, and hands each answer to the one who
// asked.
package inbox

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"uuid"
)

// ErrNotFound means the question has been answered already, or its asker has gone.
var ErrNotFound = errors.New("no such question")

// Inbox holds the questions that wait for an answer. The zero value is an empty inbox, safe for
// use by several goroutines at once.
type Inbox struct {
	// Notify, when set, is called with each new question once it's pending, before Ask waits. It
	// must not block. Set it before the inbox is used.
	Notify func(Question)

	mu      sync.Mutex
	waiting []*waiter // oldest first
}

type waiter struct {
	question Question
	answer   chan Answer
}

// Ask adds the question and waits until it's answered or ctx ends. Either way, the question
// leaves the inbox.
func (box *Inbox) Ask(ctx context.Context, question Question) (Answer, error) {
	if err := question.check(); err != nil {
		return Answer{}, err
	}

	question.ID = uuid.New()
	pending := &waiter{question: question, answer: make(chan Answer, 1)}

	box.add(pending)
	defer box.remove(pending)

	if box.Notify != nil {
		box.Notify(question)
	}

	select {
	case answer := <-pending.answer:
		return answer, nil
	case <-ctx.Done():
		return Answer{}, fmt.Errorf("wait for the answer: %w", context.Cause(ctx))
	}
}

// Pending lists the questions that wait for an answer, newest first.
func (box *Inbox) Pending() []Question {
	box.mu.Lock()
	defer box.mu.Unlock()

	questions := make([]Question, 0, len(box.waiting))
	for _, pending := range slices.Backward(box.waiting) {
		questions = append(questions, pending.question)
	}

	return questions
}

// Answer hands the answer to the question's asker, and takes the question out of the inbox.
func (box *Inbox) Answer(id uuid.UUID, answer Answer) error {
	box.mu.Lock()
	defer box.mu.Unlock()

	index := slices.IndexFunc(box.waiting, func(w *waiter) bool { return w.question.ID == id })
	if index < 0 {
		return ErrNotFound
	}

	pending := box.waiting[index]
	answer.Text = strings.TrimSpace(answer.Text)

	if err := pending.question.accepts(answer); err != nil {
		return err
	}

	box.waiting = slices.Delete(box.waiting, index, index+1)

	pending.answer <- answer // never blocks: the channel holds one answer, and it gets only one

	return nil
}

func (box *Inbox) add(pending *waiter) {
	box.mu.Lock()
	defer box.mu.Unlock()

	box.waiting = append(box.waiting, pending)
}

func (box *Inbox) remove(pending *waiter) {
	box.mu.Lock()
	defer box.mu.Unlock()

	box.waiting = slices.DeleteFunc(box.waiting, func(w *waiter) bool { return w == pending })
}
