package vapid_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/vapid"
)

// A valid private key, 32 bytes of 1, and its public half.
const (
	stored       = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE"
	storedPublic = "BG_wO5SSQc4drdQ1GeaWDgqFtBppoFwygQOqK84VlMoWPE91OlW_AdxT9sCwx-7ni0DG_30lqW4igrmJzvccFEo"
)

// fake returns a Keychain that runs the stand-in security in testdata, in mode, with item stored
// unless it is empty. It also returns the stand-in's state: the file "item" holds its one item,
// and "args" logs the arguments it ran with.
func fake(t *testing.T, mode, item string) (vapid.Keychain, *os.Root) {
	t.Helper()

	dir := t.TempDir()

	state, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = state.Close() })

	if item != "" {
		if err := state.WriteFile("item", []byte(item+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	program, err := filepath.Abs(filepath.Join("testdata", "security"))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FAKE_ITEM", filepath.Join(dir, "item"))
	t.Setenv("FAKE_ARGS", filepath.Join(dir, "args"))
	t.Setenv("FAKE_MODE", mode)

	return vapid.NewKeychain(program), state
}

// read returns what the file name in state holds, or "" if there is no such file.
func read(t *testing.T, state *os.Root, name string) string {
	t.Helper()

	data, err := state.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}

	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

// adds counts the runs of security -i, which add the item.
func adds(t *testing.T, state *os.Root) int {
	t.Helper()

	return strings.Count("\n"+read(t, state, "args"), "\n-i\n")
}

// assertPair fails unless key's halves are a P-256 key pair, base64url-encoded without padding.
func assertPair(t *testing.T, key vapid.Key) {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(key.Private)
	if err != nil {
		t.Fatalf("the private key %q: %v", key.Private, err)
	}

	private, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), raw)
	if err != nil {
		t.Fatal(err)
	}

	public, err := private.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	if want := base64.RawURLEncoding.EncodeToString(public); key.Public != want {
		t.Errorf("the public key = %q, want %q", key.Public, want)
	}
}

func TestLoadMakesAKeyOnce(t *testing.T) {
	keychain, state := fake(t, "", "")

	made, err := keychain.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	assertPair(t, made)

	if got := read(t, state, "item"); got != made.Private+"\n" {
		t.Errorf("the Keychain holds %q, want %q", got, made.Private)
	}

	again, err := keychain.Load(t.Context())
	if err != nil || again != made {
		t.Errorf("Load() again = %+v, %v; want %+v", again, err, made)
	}

	if args := read(t, state, "args"); strings.Contains(args, made.Private) {
		t.Errorf("security ran with the key in its arguments:\n%s", args)
	}

	if n := adds(t, state); n != 1 {
		t.Errorf("security -i ran %d times, want once", n)
	}
}

func TestLoadKeepsAStoredKey(t *testing.T) {
	keychain, state := fake(t, "", stored)

	key, err := keychain.Load(t.Context())
	if want := (vapid.Key{Private: stored, Public: storedPublic}); err != nil || key != want {
		t.Errorf("Load() = %+v, %v; want %+v", key, err, want)
	}

	if n := adds(t, state); n != 0 {
		t.Errorf("security -i ran %d times, want never", n)
	}
}

func TestLoadRefusesAnInvalidKey(t *testing.T) {
	tests := []struct{ name, item string }{
		{"not base64url", "AQEB+/AQ"},
		{"padded", stored + "="},
		{"too short", base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 31))},
		{"zero", base64.RawURLEncoding.EncodeToString(make([]byte, 32))},
		{"over the order", base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 32))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keychain, state := fake(t, "", test.item)

			_, err := keychain.Load(t.Context())

			want := "the VAPID key in the Keychain (service nudge, account vapid-private-key) " +
				"is invalid: "
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Errorf("Load() = %v, want %q", err, want)
			}

			if got := read(t, state, "item"); got != test.item+"\n" || adds(t, state) != 0 {
				t.Errorf("the Keychain holds %q, want %q kept", got, test.item)
			}
		})
	}
}

func TestLoadReportsWhatFailed(t *testing.T) {
	t.Run("security fails", func(t *testing.T) {
		keychain, _ := fake(t, "fail", stored)

		_, err := keychain.Load(t.Context())

		want := "read the VAPID key from the Keychain (service nudge, account vapid-private-key): " +
			"exit status 51: security: boom"
		if err == nil || err.Error() != want {
			t.Errorf("Load() = %v, want %q", err, want)
		}
	})

	t.Run("no security", func(t *testing.T) {
		keychain := vapid.NewKeychain(filepath.Join(t.TempDir(), "security"))

		if _, err := keychain.Load(t.Context()); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Load() = %v, want %v", err, fs.ErrNotExist)
		}
	})
}

func TestLoadChecksTheNewKey(t *testing.T) {
	tests := []struct{ mode, want string }{
		{
			"refuse",
			"add the VAPID key to the Keychain (service nudge, account vapid-private-key): " +
				"exit status 45: security: SecKeychainItemCreateFromContent: " +
				"The specified item already exists in the keychain.",
		},
		{"drop", "read back the new VAPID key: " +
			"the Keychain has no VAPID key (service nudge, account vapid-private-key)"},
		{"swap", "read back the new VAPID key: " +
			"the Keychain (service nudge, account vapid-private-key) holds another"},
	}

	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			keychain, _ := fake(t, test.mode, "")

			if _, err := keychain.Load(t.Context()); err == nil || err.Error() != test.want {
				t.Errorf("Load() = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadStopsWithItsContext(t *testing.T) {
	keychain, _ := fake(t, "", stored)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := keychain.Load(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Load() = %v, want %v", err, context.Canceled)
	}
}
