// Package subscriptions keeps the devices that get nudge's Web Pushes: their browsers' push
// subscriptions, in memory and in a file that outlasts a restart.
package subscriptions

import (
	"crypto/ecdh"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
)

// authSize is the length of a subscription's authentication secret (RFC 8291 §3.2).
const authSize = 16

// ErrInvalid reports a subscription that pushes can't use.
var ErrInvalid = errors.New("invalid subscription")

// Subscription is a browser's push subscription, with the names that its toJSON gives it: where
// to send the pushes, and the keys that encrypt them.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
}

// Keys are a subscription's keys, each in base64url without padding.
type Keys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// check returns ErrInvalid, with the reason, unless the endpoint is an https URL, P256dh a P-256
// public key and Auth 16 bytes.
func (s Subscription) check() error {
	endpoint, err := url.Parse(s.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" {
		return fmt.Errorf("%w: the endpoint %q isn't an https URL", ErrInvalid, s.Endpoint)
	}

	if err := checkPublicKey(s.Keys.P256dh); err != nil {
		return fmt.Errorf("%w: p256dh isn't a P-256 public key: %w", ErrInvalid, err)
	}

	auth, err := base64.RawURLEncoding.DecodeString(s.Keys.Auth)
	if err != nil || len(auth) != authSize {
		return fmt.Errorf("%w: auth isn't %d bytes in base64url", ErrInvalid, authSize)
	}

	return nil
}

// checkPublicKey checks that key is an uncompressed P-256 point in base64url.
func checkPublicKey(key string) error {
	point, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	_, err = ecdh.P256().NewPublicKey(point)

	return err
}
