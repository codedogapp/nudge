package phonedoor

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/codedogapp/nudge/internal/subscriptions"
)

// maxSubscription caps a subscription's body: its endpoint and its two keys take a few hundred
// bytes.
const maxSubscription = 4 << 10

// subscribe keeps the browser's push subscription, so its device gets the pushes.
func (s site) subscribe(w http.ResponseWriter, r *http.Request) {
	var sub subscriptions.Subscription

	body := http.MaxBytesReader(w, r.Body, maxSubscription)
	if err := json.UnmarshalRead(body, &sub); err != nil {
		http.Error(w, "read the subscription: "+err.Error(), http.StatusBadRequest)
		return
	}

	err := s.devices.Add(sub)

	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, subscriptions.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
