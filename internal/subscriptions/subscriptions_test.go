package subscriptions_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/subscriptions"
)

// The keys of RFC 8291, Appendix A: the user agent's, and the application server's public key as a
// second valid P-256 point.
const (
	p256dh      = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	auth        = "BTBZMqHH6r4Tts7J_aSIgg"
	otherP256dh = "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8"
	otherAuth   = "AAAAAAAAAAAAAAAAAAAAAA"
)

// valid returns a valid subscription to endpoint.
func valid(endpoint string) subscriptions.Subscription {
	return subscriptions.Subscription{
		Endpoint: endpoint,
		Keys:     subscriptions.Keys{P256dh: p256dh, Auth: auth},
	}
}

func encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func TestAddRefusesAnInvalidSubscription(t *testing.T) {
	point, err := base64.RawURLEncoding.DecodeString(p256dh)
	if err != nil {
		t.Fatal(err)
	}

	compressed := append([]byte{0x02 | point[64]&1}, point[1:33]...)
	offCurve := bytes.Clone(point)
	offCurve[64] ^= 1

	withKeys := func(key, secret string) subscriptions.Subscription {
		sub := valid("https://push.example/send/1")
		sub.Keys = subscriptions.Keys{P256dh: key, Auth: secret}

		return sub
	}

	tests := []struct {
		name   string
		sub    subscriptions.Subscription
		reason string
	}{
		{"no endpoint", valid(""), "endpoint"},
		{"an http endpoint", valid("http://push.example/send/1"), "endpoint"},
		{"an endpoint without a host", valid("https:///send/1"), "endpoint"},
		{"a relative endpoint", valid("/send/1"), "endpoint"},
		{"an endpoint that isn't a URL", valid("https://push.example/%zz"), "endpoint"},
		{"no p256dh", withKeys("", auth), "p256dh"},
		{"a padded p256dh", withKeys(p256dh+"=", auth), "p256dh"},
		{"a p256dh in standard base64", withKeys(std(p256dh), auth), "p256dh"},
		{"a compressed p256dh", withKeys(encode(compressed), auth), "p256dh"},
		{"a short p256dh", withKeys(encode(point[:64]), auth), "p256dh"},
		{"a p256dh off the curve", withKeys(encode(offCurve), auth), "p256dh"},
		{"no auth", withKeys(p256dh, ""), "auth"},
		{"a 15-byte auth", withKeys(p256dh, encode(make([]byte, 15))), "auth"},
		{"a 17-byte auth", withKeys(p256dh, encode(make([]byte, 17))), "auth"},
		{"an auth in standard base64", withKeys(p256dh, std(auth)), "auth"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := pathIn(t)

			err := open(t, path).Add(test.sub)
			if !errors.Is(err, subscriptions.ErrInvalid) ||
				!strings.Contains(err.Error(), test.reason) {
				t.Errorf(
					"Add() = %v, want %v naming %s",
					err,
					subscriptions.ErrInvalid,
					test.reason,
				)
			}

			assertMissing(t, path)
		})
	}
}

// std turns base64url into standard base64.
func std(key string) string {
	return strings.NewReplacer("-", "+", "_", "/").Replace(key)
}
