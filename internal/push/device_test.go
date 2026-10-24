package push_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/codedogapp/nudge/internal/subscriptions"
	"github.com/codedogapp/nudge/internal/vapid"
)

// The sizes of a push's keys (RFC 8291 §3) and of its record header (RFC 8188 §2.1): the salt,
// the record size and the length of the sender's key, which follows it.
const (
	authSize   = 16
	ikmSize    = 32
	cekSize    = 16
	nonceSize  = 12
	saltSize   = 16
	headerSize = saltSize + 4 + 1
	delimiter  = "\x02"
)

// encode writes bytes in base64url without padding, as browsers and nudge write keys.
var encode = base64.RawURLEncoding.EncodeToString

// noError ends the test when err isn't nil.
func noError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func newKey(t *testing.T) *ecdh.PrivateKey {
	t.Helper()

	key, err := ecdh.P256().GenerateKey(rand.Reader)
	noError(t, err)

	return key
}

// newVAPIDKey returns a new VAPID key, in the form nudge keeps it.
func newVAPIDKey(t *testing.T) vapid.Key {
	t.Helper()

	key := newKey(t)

	return vapid.Key{Private: encode(key.Bytes()), Public: encode(key.PublicKey().Bytes())}
}

// device is a browser's half of a push subscription: the keys that decrypt its pushes.
type device struct {
	key  *ecdh.PrivateKey
	auth []byte
}

func newDevice(t *testing.T) device {
	t.Helper()

	auth := make([]byte, authSize)
	_, err := rand.Read(auth)
	noError(t, err)

	return device{key: newKey(t), auth: auth}
}

// at returns the device's subscription, with its pushes going to endpoint.
func (d device) at(endpoint string) subscriptions.Subscription {
	return subscriptions.Subscription{
		Endpoint: endpoint,
		Keys: subscriptions.Keys{
			P256dh: encode(d.key.PublicKey().Bytes()),
			Auth:   encode(d.auth),
		},
	}
}

// decrypt opens a push's body as the browser does (RFC 8291 §3.4, RFC 8188 §2): one record,
// whose header holds the salt and the sender's one-time key, and whose plaintext ends in the
// delimiter and zeros.
func (d device) decrypt(t *testing.T, body []byte) string {
	t.Helper()

	if len(body) < headerSize || len(body) < headerSize+int(body[headerSize-1]) {
		t.Fatalf("the body has %d bytes, too few for a record", len(body))
	}

	start := headerSize + int(body[headerSize-1])
	salt, senderKey := body[:saltSize], body[headerSize:start]
	sender, err := ecdh.P256().NewPublicKey(senderKey)
	noError(t, err)

	secret, err := d.key.ECDH(sender)
	noError(t, err)

	info := "WebPush: info\x00" + string(d.key.PublicKey().Bytes()) + string(senderKey)
	ikm, err := hkdf.Key(sha256.New, secret, d.auth, info, ikmSize)
	noError(t, err)

	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", cekSize)
	noError(t, err)

	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", nonceSize)
	noError(t, err)

	block, err := aes.NewCipher(cek)
	noError(t, err)

	gcm, err := cipher.NewGCM(block)
	noError(t, err)

	plain, err := gcm.Open(nil, nonce, body[start:], nil)
	noError(t, err)

	message, ok := bytes.CutSuffix(bytes.TrimRight(plain, "\x00"), []byte(delimiter))
	if !ok {
		t.Fatal("the plaintext doesn't end in the delimiter and zeros")
	}

	return string(message)
}
