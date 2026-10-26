package setup_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/codedogapp/nudge/internal/setup"
)

func TestPublishServesThePhoneDoor(t *testing.T) {
	var got []string

	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		got = append([]string{name}, args...)
		return nil, nil
	}

	if _, err := setup.Publish(t.Context(), run); err != nil {
		t.Fatal(err)
	}

	want := []string{"tailscale", "serve", "--bg", "--https=8443", "8700"}
	if !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestPublishFailsWithTheToolsError(t *testing.T) {
	_, err := setup.Publish(t.Context(), answering("", errors.New("denied")))
	if err == nil {
		t.Error("got no error")
	}
}
