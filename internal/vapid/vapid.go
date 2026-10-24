// Package vapid keeps the server's VAPID key, which signs its Web Pushes, in the login Keychain.
package vapid

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// The Keychain item that holds the private key, as errors name it, and the exit code of security
// when the item isn't there (errSecItemNotFound).
const (
	service  = "nudge"
	account  = "vapid-private-key"
	item     = "service " + service + ", account " + account
	notFound = 44
)

// errMissing reports that the Keychain has no key yet.
var errMissing = errors.New("the Keychain has no VAPID key (" + item + ")")

// Key is a VAPID key pair, each half base64url-encoded without padding, as Web Push wants it.
type Key struct {
	Private string
	Public  string
}

// Load returns the VAPID key in the login Keychain. When there is none, it makes one and adds it
// there first. It never replaces a stored key.
func Load(ctx context.Context) (Key, error) {
	stored, err := find(ctx)
	if errors.Is(err, errMissing) {
		return create(ctx)
	}

	if err != nil {
		return Key{}, err
	}

	return parse(stored)
}

// create makes a key, adds it to the Keychain, and reads it back to check that it is there.
func create(ctx context.Context) (Key, error) {
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return Key{}, fmt.Errorf("make a VAPID key: %w", err)
	}

	made := base64.RawURLEncoding.EncodeToString(private.Bytes())
	if err := add(ctx, made); err != nil {
		return Key{}, err
	}

	stored, err := find(ctx)
	if err != nil {
		return Key{}, fmt.Errorf("read back the new VAPID key: %w", err)
	}

	if stored != made {
		return Key{}, fmt.Errorf(
			"read back the new VAPID key: the Keychain (%s) holds another",
			item,
		)
	}

	return parse(stored)
}

// parse checks that the stored private key is one, and derives its public half.
func parse(stored string) (Key, error) {
	raw, err := base64.RawURLEncoding.DecodeString(stored)
	if err != nil {
		return Key{}, invalid(err)
	}

	private, err := ecdh.P256().NewPrivateKey(raw)
	if err != nil {
		return Key{}, invalid(err)
	}

	public := base64.RawURLEncoding.EncodeToString(private.PublicKey().Bytes())

	return Key{Private: stored, Public: public}, nil
}

// invalid reports a stored key that isn't a P-256 private key in base64url without padding.
func invalid(err error) error {
	return fmt.Errorf("the VAPID key in the Keychain (%s) is invalid: %w", item, err)
}

// find returns the stored private key, or errMissing.
func find(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx,
		"security", "find-generic-password", "-s", service, "-a", account, "-w")

	out, err := run(cmd, "")
	if exit, ok := errors.AsType[*exec.ExitError](err); ok && exit.ExitCode() == notFound {
		return "", errMissing
	}

	if err != nil {
		return "", fmt.Errorf("read the VAPID key from the Keychain (%s): %w", item, err)
	}

	return strings.TrimSuffix(out, "\n"), nil
}

// add adds the private key to the Keychain. security reads the command from stdin, so the key
// never shows in a process's arguments. Without -U, it refuses to replace an item.
func add(ctx context.Context, private string) error {
	command := fmt.Sprintf("add-generic-password -s %s -a %s -w %s\n", service, account, private)
	if _, err := run(exec.CommandContext(ctx, "security", "-i"), command); err != nil {
		return fmt.Errorf("add the VAPID key to the Keychain (%s): %w", item, err)
	}

	return nil
}

// run runs cmd, macOS's security tool found on PATH, with stdin, and returns what it prints. Its
// error carries what the tool printed on stderr.
func run(cmd *exec.Cmd, stdin string) (string, error) {
	var stdout, stderr strings.Builder

	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if printed := strings.TrimSpace(stderr.String()); printed != "" {
			return "", fmt.Errorf("%w: %s", err, printed)
		}

		return "", err
	}

	return stdout.String(), nil
}
