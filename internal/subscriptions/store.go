package subscriptions

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Only the user can read and write the subscriptions.
const (
	privateFolder fs.FileMode = 0o700
	privateFile   fs.FileMode = 0o600
)

// Store keeps the subscriptions in memory and in their file, which it writes on each change.
type Store struct {
	path string

	mu   sync.Mutex
	list []Subscription
}

// Open returns the store kept in the file at path, and makes the file's folder when it's missing.
// A missing file holds no subscriptions.
func Open(path string) (*Store, error) {
	store := &Store{path: path}

	root, err := store.root()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	data, err := root.ReadFile(filepath.Base(path))
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read the subscriptions file %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &store.list); err != nil {
		return nil, store.invalid(err)
	}

	for _, sub := range store.list {
		if err := sub.check(); err != nil {
			return nil, store.invalid(err)
		}
	}

	return store, nil
}

// Add adds sub, or puts it in the place of the subscription with its endpoint, and writes the
// file. An unchanged subscription writes nothing. When the write fails, the store stays as it was.
func (s *Store) Add(sub Subscription) error {
	if err := sub.check(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sameEndpoint := func(old Subscription) bool { return old.Endpoint == sub.Endpoint }

	index := slices.IndexFunc(s.list, sameEndpoint)
	if index >= 0 && s.list[index] == sub {
		return nil
	}

	next := slices.Clone(s.list)
	if index < 0 {
		next = append(next, sub)
	} else {
		next[index] = sub
	}

	if err := s.save(next); err != nil {
		return fmt.Errorf("write the subscriptions file %s: %w", s.path, err)
	}

	s.list = next

	return nil
}

// invalid reports a file that isn't a list of valid subscriptions.
func (s *Store) invalid(err error) error {
	return fmt.Errorf("the subscriptions file %s is invalid: %w", s.path, err)
}

// root opens the file's folder, and makes it first when it's missing.
func (s *Store) root() (*os.Root, error) {
	folder := filepath.Dir(s.path)
	if err := os.MkdirAll(folder, privateFolder); err != nil {
		return nil, fmt.Errorf("make the subscriptions folder: %w", err)
	}

	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("open the subscriptions folder: %w", err)
	}

	return root, nil
}

// save writes list to a new temporary file next to the file, then renames it over the file, so a
// crash leaves the old list or the new one, never part of one.
func (s *Store) save(list []Subscription) error {
	data, err := json.Marshal(list, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}

	data = append(data, '\n')

	root, err := s.root()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	name := filepath.Base(s.path)
	temp := name + ".tmp"

	// A temporary file that a crash left behind goes first, so the new one gets mode 0600.
	_ = root.Remove(temp)

	if err := write(root, temp, data); err != nil {
		return err
	}

	return root.Rename(temp, name)
}

// write writes data to a new file name in root, with mode 0600, and syncs it to disk.
func write(root *os.Root, name string, data []byte) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateFile)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}

	return errors.Join(err, file.Close())
}
