package push_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
)

// payload is what a push carries, as the service worker reads it.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// sendOne sends question to one device, and returns the payload that the device decrypts.
func sendOne(t *testing.T, question inbox.Question) payload {
	t.Helper()

	service := newPushService(t)
	phone := newDevice(t)
	sender, _, logs := newSender(t, service, phone.at(service.endpoint("201", "phone")))

	sender.Send(t.Context(), question)

	got := service.pushes()
	if len(got) != 1 {
		t.Fatalf("the push service got %d pushes, want 1; log:\n%s", len(got), logs)
	}

	assertLog(t, logs)

	var message payload
	noError(t, json.Unmarshal([]byte(phone.decrypt(t, got[0].body)), &message))

	return message
}

func TestSendPushesTheQuestionToEachDevice(t *testing.T) {
	service := newPushService(t)
	devices := []device{newDevice(t), newDevice(t)}
	sender, _, logs := newSender(t, service,
		devices[0].at(service.endpoint("201", "first")),
		devices[1].at(service.endpoint("201", "second")),
	)

	sender.Send(t.Context(), inbox.Question{Project: "atlas", Text: "Merge it?"})

	got := service.pushes()
	if len(got) != len(devices) {
		t.Fatalf("the push service got %d pushes, want %d", len(got), len(devices))
	}

	for number, device := range devices {
		message := device.decrypt(t, got[number].body)
		if want := `{"title":"atlas","body":"Merge it?"}`; message != want {
			t.Errorf("push %d = %s, want %s", number, message, want)
		}

		assertHeaders(t, got[number].header, sender.Key, service.server.URL)
	}

	assertLog(t, logs)
}

func TestSendTitlesAQuestionWithoutAProjectNudge(t *testing.T) {
	got := sendOne(t, inbox.Question{Text: "Merge it?"})

	if want := (payload{Title: "nudge", Body: "Merge it?"}); got != want {
		t.Errorf("payload = %+v, want %+v", got, want)
	}
}

func TestSendCutsALongTitleAndBody(t *testing.T) {
	tests := []struct {
		name     string
		question inbox.Question
		want     payload
	}{
		{
			name: "at the limits",
			question: inbox.Question{
				Project: strings.Repeat("p", 100),
				Text:    strings.Repeat("é", 500),
			},
			want: payload{Title: strings.Repeat("p", 100), Body: strings.Repeat("é", 500)},
		},
		{
			name: "over the limits",
			question: inbox.Question{
				Project: strings.Repeat("p", 101),
				Text:    strings.Repeat("é", 501),
			},
			want: payload{
				Title: strings.Repeat("p", 99) + "…",
				Body:  strings.Repeat("é", 499) + "…",
			},
		},
		{
			// Each control character is escaped as \u00XX: six bytes, the most a character takes.
			name: "the longest payload",
			question: inbox.Question{
				Project: strings.Repeat("\x01", 200),
				Text:    strings.Repeat("\x01", 1000),
			},
			want: payload{
				Title: strings.Repeat("\x01", 99) + "…",
				Body:  strings.Repeat("\x01", 499) + "…",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sendOne(t, test.question); got != test.want {
				t.Errorf("payload = %+q, want %+q", got, test.want)
			}
		})
	}
}

func TestSendDropsAQuestionThatIsntUTF8(t *testing.T) {
	service := newPushService(t)
	sender, _, logs := newSender(t, service, newDevice(t).at(service.endpoint("201", "phone")))

	sender.Send(t.Context(), inbox.Question{Project: "atlas", Text: "\xff"})

	if got := service.pushes(); len(got) != 0 {
		t.Errorf("the push service got %d pushes, want none", len(got))
	}

	assertLog(t, logs, map[string]any{"level": "WARN", "msg": "push failed", "err": anyError})
}
