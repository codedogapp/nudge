package push_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/nudge/internal/vapid"
)

// claims are a VAPID JWT's claims (RFC 8292 §2).
type claims struct {
	Aud string `json:"aud"`
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

// assertHeaders checks a push's headers: a day to live, high urgency, aes128gcm, and a VAPID
// Authorization from key, for the push service at audience.
func assertHeaders(t *testing.T, header http.Header, key vapid.Key, audience string) {
	t.Helper()

	for name, want := range map[string]string{
		"TTL":              "86400",
		"Urgency":          "high",
		"Content-Encoding": "aes128gcm",
	} {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	assertVAPID(t, header.Get("Authorization"), key, audience)
}

// assertVAPID checks an Authorization (RFC 8292 §3): k is the key's public half, and t a JWT that
// the key signed, from nudge's repository, for audience, which ends within a day.
func assertVAPID(t *testing.T, authorization string, key vapid.Key, audience string) {
	t.Helper()

	token, public, ok := strings.Cut(strings.TrimPrefix(authorization, "vapid t="), ", k=")
	if !ok || !strings.HasPrefix(authorization, "vapid t=") {
		t.Fatalf("Authorization = %q, want vapid t=…, k=…", authorization)
	}

	if public != key.Public {
		t.Errorf("k = %q, want the key's public half %q", public, key.Public)
	}

	got := verify(t, token, key)
	if got.Sub != "https://github.com/codedogapp/nudge" || got.Aud != audience {
		t.Errorf("sub, aud = %q, %q; want nudge's repository, %q", got.Sub, got.Aud, audience)
	}

	if left := time.Until(time.Unix(got.Exp, 0)); left <= 0 || left > 24*time.Hour {
		t.Errorf("exp is in %v, want within a day", left)
	}
}

// verify checks that key signed the JWT with ES256 (RFC 7518 §3.4), and returns its claims.
func verify(t *testing.T, token string, key vapid.Key) claims {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("t = %q, want a JWT", token)
	}

	var header struct {
		Alg string `json:"alg"`
	}
	noError(t, json.Unmarshal(decode(t, parts[0]), &header))

	public, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), decode(t, key.Public))
	noError(t, err)

	signature := decode(t, parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:len(signature)/2])
	s := new(big.Int).SetBytes(signature[len(signature)/2:])

	if header.Alg != "ES256" || len(signature) != 64 || !ecdsa.Verify(public, digest[:], r, s) {
		t.Errorf("the JWT isn't signed by the key with ES256: alg %q", header.Alg)
	}

	var got claims
	noError(t, json.Unmarshal(decode(t, parts[1]), &got))

	return got
}

// decode reads base64url without padding.
func decode(t *testing.T, text string) []byte {
	t.Helper()

	data, err := base64.RawURLEncoding.DecodeString(text)
	noError(t, err)

	return data
}
