package vapid

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// fileMode lets only the user read the key.
const fileMode = 0o600

// File keeps the key in a file, for a host with no Keychain, such as a container. The file holds
// the private key, base64url-encoded without padding.
type File struct {
	path string
}

// NewFile returns a File at path.
func NewFile(path string) File {
	return File{path: path}
}

// Load returns the VAPID key in the file. When there is no file, it makes a key and writes it
// there first. It never replaces a key.
func (f File) Load(context.Context) (Key, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return f.create()
	}

	if err != nil {
		return Key{}, fmt.Errorf("read the VAPID key file %s: %w", f.path, err)
	}

	key, err := derive(strings.TrimSpace(string(data)))
	if err != nil {
		return Key{}, fmt.Errorf("the VAPID key file %s is invalid: %w", f.path, err)
	}

	return key, nil
}

// create makes a key and writes it to the file, which isn't there, in a folder that is.
func (f File) create() (Key, error) {
	made, err := generate()
	if err != nil {
		return Key{}, err
	}

	if err := os.WriteFile(f.path, []byte(made+"\n"), fileMode); err != nil {
		return Key{}, fmt.Errorf("write the VAPID key file %s: %w", f.path, err)
	}

	return derive(made)
}
