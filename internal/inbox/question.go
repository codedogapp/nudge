package inbox

import (
	"errors"
	"slices"
	"strings"
	"uuid"
)

var (
	// ErrNoText means the question has no text.
	ErrNoText = errors.New("the question has no text")
	// ErrBadOptions means one of the question's options is empty or repeated.
	ErrBadOptions = errors.New("an option is empty or repeated")
	// ErrEmptyAnswer means the answer has neither an option nor a text.
	ErrEmptyAnswer = errors.New("the answer has neither an option nor a text")
	// ErrUnknownOption means the answer picks an option that the question doesn't offer.
	ErrUnknownOption = errors.New("the question doesn't offer the answer's option")
)

// Question is what an agent asks. The inbox gives it its ID.
type Question struct {
	ID      uuid.UUID
	Project string
	Text    string
	Details string
	Options []string
}

// Answer is the user's reply: one of the question's options, a text, or both.
type Answer struct {
	Option string
	Text   string
}

func (q Question) check() error {
	if strings.TrimSpace(q.Text) == "" {
		return ErrNoText
	}

	for i, option := range q.Options {
		if strings.TrimSpace(option) == "" || slices.Contains(q.Options[:i], option) {
			return ErrBadOptions
		}
	}

	return nil
}

func (q Question) accepts(answer Answer) error {
	if answer.Option == "" && answer.Text == "" {
		return ErrEmptyAnswer
	}

	if answer.Option != "" && !slices.Contains(q.Options, answer.Option) {
		return ErrUnknownOption
	}

	return nil
}
