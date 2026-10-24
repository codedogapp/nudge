package push_test

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/codedogapp/nudge/internal/push"
	"github.com/codedogapp/nudge/internal/subscriptions"
)

// answer is what the fake push service says when it refuses a push: a reason, as Apple's push
// service gives, and more than nudge logs.
var answer = `{"reason":"BadJwtToken"}` + strings.Repeat(" ", 600)

// pushed is a push that the fake push service got.
type pushed struct {
	path   string
	header http.Header
	body   []byte
}

// pushService is a fake push service, over TLS. It answers a push to /push/<status>/<name> with
// that status, drops the connection when the status isn't a number, and keeps each push it got.
type pushService struct {
	server *httptest.Server
	mu     sync.Mutex
	got    []pushed
}

func newPushService(t *testing.T) *pushService {
	t.Helper()

	service := &pushService{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /push/{status}/{name}", service.receive)
	service.server = httptest.NewTLSServer(mux)
	t.Cleanup(service.server.Close)

	return service
}

// endpoint returns the endpoint of a device whose pushes get status.
func (p *pushService) endpoint(status, name string) string {
	return p.server.URL + "/push/" + status + "/" + name
}

// host is the push service's host, as nudge's log names it.
func (p *pushService) host() string {
	return p.server.Listener.Addr().String()
}

// pushes returns the pushes that the service got, in the order they came.
func (p *pushService) pushes() []pushed {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.got)
}

func (p *pushService) receive(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	p.mu.Lock()
	p.got = append(p.got, pushed{path: r.URL.Path, header: r.Header.Clone(), body: body})
	p.mu.Unlock()

	status, err := strconv.Atoi(r.PathValue("status"))
	if err != nil {
		panic(http.ErrAbortHandler)
	}

	w.WriteHeader(status)

	if status >= http.StatusMultipleChoices {
		_, _ = io.WriteString(w, answer)
	}
}

// newSender returns a sender with a new key, which pushes through the service's client to the
// devices; the path of the devices' file; and the sender's log, in JSON.
func newSender(
	t *testing.T,
	service *pushService,
	devices ...subscriptions.Subscription,
) (*push.Sender, string, *bytes.Buffer) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "nudge", "subscriptions.json")
	store, err := subscriptions.Open(path)
	noError(t, err)

	for _, device := range devices {
		noError(t, store.Add(device))
	}

	logs := &bytes.Buffer{}
	sender := &push.Sender{
		Key:     newVAPIDKey(t),
		Devices: store,
		Client:  service.server.Client(),
		Logger:  slog.New(slog.NewJSONHandler(logs, nil)),
	}

	return sender, path, logs
}

// anyError stands for an error's text in a log record: its wording comes from elsewhere, so
// assertLog only checks that there is one.
const anyError = "<error>"

// assertLog checks the log's records, but for their time, and that it never names an endpoint.
func assertLog(t *testing.T, logs *bytes.Buffer, want ...map[string]any) {
	t.Helper()

	text := logs.String()
	if strings.Contains(text, "/push/") {
		t.Errorf("the log names an endpoint:\n%s", text)
	}

	var records []map[string]any

	for line := range strings.Lines(text) {
		var record map[string]any
		noError(t, json.Unmarshal([]byte(line), &record))
		delete(record, slog.TimeKey)

		if err, ok := record["err"].(string); ok && err != "" {
			record["err"] = anyError
		}

		records = append(records, record)
	}

	if !reflect.DeepEqual(records, want) {
		t.Errorf("log records = %v\nwant %v", records, want)
	}
}

// assertDevices checks the devices that a store lists.
func assertDevices(
	t *testing.T,
	got []subscriptions.Subscription,
	want ...subscriptions.Subscription,
) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("devices = %v, want %v", got, want)
	}
}
