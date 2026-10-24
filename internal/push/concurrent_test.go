package push_test

import (
	"strings"
	"sync"
	"testing"
)

func TestConcurrentSendsPushToEachDeviceAndForgetAGoneOne(t *testing.T) {
	const sends = 5

	service := newPushService(t)
	kept := newDevice(t).at(service.endpoint("201", "kept"))
	sender, _, logs := newSender(t, service, newDevice(t).at(service.endpoint("410", "gone")), kept)

	var group sync.WaitGroup
	for range sends {
		group.Go(func() { sender.Send(t.Context(), question) })
	}

	group.Wait()

	assertDevices(t, sender.Devices.List(), kept)

	pushes := 0

	for _, got := range service.pushes() {
		if got.path == "/push/201/kept" {
			pushes++
		}
	}

	if pushes != sends {
		t.Errorf("the kept device got %d pushes, want %d", pushes, sends)
	}

	text := logs.String()
	if !strings.Contains(text, `"msg":"forgot a device"`) ||
		strings.Contains(text, `"level":"WARN"`) {
		t.Errorf("the log doesn't say the device was forgotten, or has a warning:\n%s", text)
	}
}
