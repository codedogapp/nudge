package push_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
	"github.com/codedogapp/nudge/internal/subscriptions"
)

var question = inbox.Question{Project: "atlas", Text: "Merge it?"}

func TestSendForgetsAGoneDevice(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusGone} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			service := newPushService(t)
			kept := newDevice(t).at(service.endpoint("201", "kept"))
			sender, path, logs := newSender(t, service,
				newDevice(t).at(service.endpoint(strconv.Itoa(status), "gone")),
				kept,
			)

			sender.Send(t.Context(), question)

			assertDevices(t, sender.Devices.List(), kept)

			reopened, err := subscriptions.Open(path)
			noError(t, err)
			assertDevices(t, reopened.List(), kept)

			assertLog(t, logs, map[string]any{
				"level":   "INFO",
				"msg":     "forgot a device",
				"service": service.host(),
				"status":  float64(status),
			})
			assertPushedLast(t, service, "/push/201/kept")
		})
	}
}

func TestSendWarnsAboutAFailedPushAndGoesOn(t *testing.T) {
	service := newPushService(t)
	devices := []subscriptions.Subscription{
		newDevice(t).at(service.endpoint("403", "refused")),
		newDevice(t).at(service.endpoint("drop", "dropped")),
		newDevice(t).at(service.endpoint("201", "kept")),
	}
	sender, _, logs := newSender(t, service, devices...)

	sender.Send(t.Context(), question)

	assertDevices(t, sender.Devices.List(), devices...)
	assertLog(t, logs,
		map[string]any{
			"level":   "WARN",
			"msg":     "push refused",
			"service": service.host(),
			"status":  float64(http.StatusForbidden),
			"reason":  answer[:512],
		},
		map[string]any{
			"level":   "WARN",
			"msg":     "push failed",
			"service": service.host(),
			"err":     anyError,
		},
	)
	assertPushedLast(t, service, "/push/201/kept")
}

func TestSendWarnsWhenItCantMakeAPush(t *testing.T) {
	service := newPushService(t)
	sender, _, logs := newSender(t, service, newDevice(t).at(service.endpoint("201", "phone")))
	sender.Key.Private = "not base64url"

	sender.Send(t.Context(), question)

	if got := service.pushes(); len(got) != 0 {
		t.Errorf("the push service got %d pushes, want none", len(got))
	}

	assertLog(t, logs, map[string]any{
		"level":   "WARN",
		"msg":     "push failed",
		"service": service.host(),
		"err":     anyError,
	})
}

func TestSendWarnsWhenItCantForgetADevice(t *testing.T) {
	service := newPushService(t)
	gone := newDevice(t).at(service.endpoint("410", "gone"))
	sender, path, logs := newSender(t, service, gone)

	// A file in the place of the folder makes writing the devices' file fail.
	folder := filepath.Dir(path)
	noError(t, os.RemoveAll(folder))
	noError(t, os.WriteFile(folder, nil, 0o600))

	sender.Send(t.Context(), question)

	assertDevices(t, sender.Devices.List(), gone)
	assertLog(t, logs, map[string]any{
		"level":   "WARN",
		"msg":     "couldn't forget a device",
		"service": service.host(),
		"err":     anyError,
	})
}

// assertPushedLast checks that the last push the service got went to path.
func assertPushedLast(t *testing.T, service *pushService, path string) {
	t.Helper()

	got := service.pushes()
	if len(got) == 0 || got[len(got)-1].path != path {
		t.Errorf("the push service's last push isn't to %s", path)
	}
}
