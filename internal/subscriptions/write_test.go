package subscriptions_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// breakFolder puts a file in the place of the folder of the file at path, so that writes fail,
// until the test calls the function it returns.
func breakFolder(t *testing.T, path string) (repair func()) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(dir, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	return func() {
		if err := os.Remove(dir); err != nil {
			t.Fatal(err)
		}

		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAddOfAnUnchangedSubscriptionWritesNothing(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)
	add(t, store, valid(first))

	breakFolder(t, path)

	if err := store.Add(valid(first)); err != nil {
		t.Errorf("Add(the same) = %v, want nil and no write", err)
	}

	if err := store.Add(valid(second)); err == nil {
		t.Error("Add(another) = nil, want the broken folder's error")
	}
}

func TestAFailedWriteKeepsTheStoreAsItWas(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)
	add(t, store, valid(first))

	repair := breakFolder(t, path)

	err := store.Add(valid(second))
	if want := "write the subscriptions file " + path; err == nil ||
		!strings.HasPrefix(err.Error(), want) {
		t.Errorf("Add() = %v, want an error that starts with %q", err, want)
	}

	repair()
	add(t, store, valid(third))
	assertList(t, path, valid(first), valid(third))
}

func TestAddReplacesAStaleTemporaryFile(t *testing.T) {
	path := pathIn(t)
	root := folder(t, path)
	stale := filepath.Base(path) + ".tmp"

	if err := root.WriteFile(stale, []byte("[{"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := root.Chmod(stale, 0o644); err != nil {
		t.Fatal(err)
	}

	add(t, open(t, path), valid(first))
	assertList(t, path, valid(first))

	if _, err := root.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(the temporary file) = %v, want it gone", err)
	}
}
