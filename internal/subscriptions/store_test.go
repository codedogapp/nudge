package subscriptions_test

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/codedogapp/nudge/internal/subscriptions"
)

const (
	first  = "https://push.example/send/1"
	second = "https://push.example/send/2"
	third  = "https://push.example/send/3"
)

// pathIn returns the path of a subscriptions file in a folder of the test's own.
func pathIn(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "subscriptions.json")
}

func open(t *testing.T, path string) *subscriptions.Store {
	t.Helper()

	store, err := subscriptions.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func add(t *testing.T, store *subscriptions.Store, sub subscriptions.Subscription) {
	t.Helper()

	if err := store.Add(sub); err != nil {
		t.Fatal(err)
	}
}

// folder opens the folder of the file at path.
func folder(t *testing.T, path string) *os.Root {
	t.Helper()

	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = root.Close() })

	return root
}

// read returns the subscriptions in the file at path, and checks that only its owner can read it.
func read(t *testing.T, path string) []subscriptions.Subscription {
	t.Helper()

	root := folder(t, path)
	name := filepath.Base(path)

	info, err := root.Stat(name)
	if err != nil {
		t.Fatal(err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("the file's mode = %#o, want 0600", mode)
	}

	data, err := root.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}

	var list []subscriptions.Subscription
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatal(err)
	}

	return list
}

func assertList(t *testing.T, path string, want ...subscriptions.Subscription) {
	t.Helper()

	if got := read(t, path); !slices.Equal(got, want) {
		t.Errorf("the file holds %+v, want %+v", got, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := folder(t, path).Stat(filepath.Base(path)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(the file) = %v, want it missing", err)
	}
}

func TestOpenMakesTheFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nudge", "subscriptions.json")
	store := open(t, path)

	info, err := os.Stat(filepath.Dir(path))
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("Stat(the folder) = %v, %v; want a folder with mode 0700", info, err)
	}

	assertMissing(t, path)
	add(t, store, valid(first))
	assertList(t, path, valid(first))
}

func TestAddReplacesInPlaceAndAppends(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)

	add(t, store, valid(first))
	add(t, store, valid(second))

	renewed := valid(first)
	renewed.Keys = subscriptions.Keys{P256dh: otherP256dh, Auth: otherAuth}
	add(t, store, renewed)
	add(t, store, valid(third))

	assertList(t, path, renewed, valid(second), valid(third))
}

func TestAReopenedStoreHasTheList(t *testing.T) {
	path := pathIn(t)
	before := open(t, path)
	add(t, before, valid(first))
	add(t, before, valid(second))

	renewed := valid(first)
	renewed.Keys = subscriptions.Keys{P256dh: otherP256dh, Auth: otherAuth}
	add(t, open(t, path), renewed)

	assertList(t, path, renewed, valid(second))
}

func TestConcurrentAddsKeepEverySubscription(t *testing.T) {
	const devices = 20

	path := pathIn(t)
	store := open(t, path)

	var adding sync.WaitGroup

	for i := range devices {
		adding.Go(func() {
			if err := store.Add(valid(fmt.Sprintf("%s/%d", first, i))); err != nil {
				t.Error(err)
			}
		})
	}

	adding.Wait()

	got := read(t, path)
	endpoints := make([]string, 0, len(got))

	for _, sub := range got {
		endpoints = append(endpoints, sub.Endpoint)
	}

	slices.Sort(endpoints)

	if endpoints = slices.Compact(endpoints); len(endpoints) != devices {
		t.Errorf("the file holds %d endpoints, want %d", len(endpoints), devices)
	}
}
