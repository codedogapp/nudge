// Package push sends each new question to the subscribed devices as a Web Push.
package push

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/subscriptions"
	"github.com/codedogapp/nudge/internal/vapid"
)

const (
	// subscriber says who sends the pushes, as the VAPID JWT's subject. Apple wants a URL or a
	// mailto: address.
	subscriber = "https://github.com/codedogapp/nudge"
	// ttl is how many seconds a push service keeps a push for a device that's offline: a day.
	ttl = 24 * 60 * 60
	// maxTitle and maxBody, in characters, keep the payload in one 4,096-byte record, even when
	// every character is escaped as \u00XX.
	maxTitle = 100
	maxBody  = 500
	// maxReason is how much of a push service's answer goes to the log.
	maxReason = 512
)

// Sender sends pushes to the subscribed devices, signed with the VAPID key.
type Sender struct {
	Key     vapid.Key
	Devices *subscriptions.Store
	Client  *http.Client
	Logger  *slog.Logger
}

// message is a push's payload, as the service worker reads it.
type message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Send sends question to each subscribed device in turn, and forgets a device whose push service
// says its subscription is gone. It logs what fails: push is best effort.
func (s *Sender) Send(ctx context.Context, question inbox.Question) {
	payload, err := json.Marshal(message{
		Title: cut(cmp.Or(question.Project, "nudge"), maxTitle),
		Body:  cut(question.Text, maxBody),
	})
	if err != nil {
		s.Logger.Warn("push failed", "err", err)
		return
	}

	for _, device := range s.Devices.List() {
		s.sendTo(ctx, device, payload)
	}
}

// sendTo sends payload to device. The log names its push service, never its endpoint: that's the
// device's address.
func (s *Sender) sendTo(ctx context.Context, device subscriptions.Subscription, payload []byte) {
	service := ""
	if endpoint, err := url.Parse(device.Endpoint); err == nil {
		service = endpoint.Host
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: device.Endpoint,
		Keys:     webpush.Keys{Auth: device.Keys.Auth, P256dh: device.Keys.P256dh},
	}, &webpush.Options{
		HTTPClient:      s.Client,
		Subscriber:      subscriber,
		TTL:             ttl,
		Urgency:         webpush.UrgencyHigh,
		VAPIDPublicKey:  s.Key.Public,
		VAPIDPrivateKey: s.Key.Private,
	})
	if err != nil {
		s.Logger.Warn("push failed", "service", service, "err", withoutURL(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusNotFound || code == http.StatusGone:
		s.forget(device.Endpoint, service, code)
	case code >= http.StatusMultipleChoices:
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, maxReason))
		s.Logger.Warn("push refused", "service", service, "status", code, "reason", string(reason))
	}
}

// forget removes the device with endpoint, whose subscription is gone.
func (s *Sender) forget(endpoint, service string, status int) {
	if err := s.Devices.Remove(endpoint); err != nil {
		s.Logger.Warn("couldn't forget a device", "service", service, "err", err)
		return
	}

	s.Logger.Info("forgot a device", "service", service, "status", status)
}

// cut shortens text to limit characters, the last of them "…", when it's longer.
func cut(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	return string([]rune(text)[:limit-1]) + "…"
}

// withoutURL drops the endpoint from the error of a request that failed.
func withoutURL(err error) error {
	if failed, ok := errors.AsType[*url.Error](err); ok {
		return failed.Err
	}

	return err
}
