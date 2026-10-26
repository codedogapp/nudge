package vapid

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Folder and file modes: only the user can open them.
const (
	folderMode = 0o700
	fileMode   = 0o600
)

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

// create makes a key and writes it to a new file.
func (f File) create() (Key, error) {
	made, err := generate()
	if err != nil {
		return Key{}, err
	}

	if err := os.MkdirAll(filepath.Dir(f.path), folderMode); err != nil {
		return Key{}, fmt.Errorf("make the folder of the VAPID key file: %w", err)
	}

	// O_EXCL: a key that appeared meanwhile is never replaced.
	file, err := os.OpenFile(f.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return Key{}, fmt.Errorf("write the VAPID key file %s: %w", f.path, err)
	}

	_, writeErr := file.WriteString(made + "\n")
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return Key{}, fmt.Errorf("write the VAPID key file %s: %w", f.path, err)
	}

	return derive(made)
}
