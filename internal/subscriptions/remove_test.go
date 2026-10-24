package subscriptions_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/subscriptions"
)

func assertStore(t *testing.T, store *subscriptions.Store, want ...subscriptions.Subscription) {
	t.Helper()

	if got := store.List(); !slices.Equal(got, want) {
		t.Errorf("List() = %+v, want %+v", got, want)
	}
}

func TestListReturnsACopyInTheOrderAdded(t *testing.T) {
	store := open(t, pathIn(t))
	add(t, store, valid(second))
	add(t, store, valid(first))

	list := store.List()
	list[0] = valid(third)

	assertStore(t, store, valid(second), valid(first))
}

func TestRemoveForgetsTheDevice(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)
	add(t, store, valid(first))
	add(t, store, valid(second))
	add(t, store, valid(third))

	if err := store.Remove(second); err != nil {
		t.Fatalf("Remove() = %v", err)
	}

	assertStore(t, store, valid(first), valid(third))
	assertList(t, path, valid(first), valid(third))
	assertStore(t, open(t, path), valid(first), valid(third))
}

func TestRemoveOfAnUnknownEndpointWritesNothing(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)
	add(t, store, valid(first))

	breakFolder(t, path)

	if err := store.Remove(second); err != nil {
		t.Errorf("Remove(unknown) = %v, want nil and no write", err)
	}

	assertStore(t, store, valid(first))
}

func TestAFailedRemoveKeepsTheStoreAsItWas(t *testing.T) {
	path := pathIn(t)
	store := open(t, path)
	add(t, store, valid(first))
	add(t, store, valid(second))

	repair := breakFolder(t, path)

	err := store.Remove(first)
	if want := "write the subscriptions file " + path; err == nil ||
		!strings.HasPrefix(err.Error(), want) {
		t.Errorf("Remove() = %v, want an error that starts with %q", err, want)
	}

	assertStore(t, store, valid(first), valid(second))
	repair()
	add(t, store, valid(third))
	assertList(t, path, valid(first), valid(second), valid(third))
}
