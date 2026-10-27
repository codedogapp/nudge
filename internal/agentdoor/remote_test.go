package agentdoor

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestARemoteClientReachesTheAgentEndpoints(t *testing.T) {
	var gotPath, gotHost string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotHost = r.URL.Path, r.Host

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client, err := newRemote(server.URL+"/base", server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://nudge/questions", nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = resp.Body.Close()

	if gotPath != "/base/agent/questions" || gotHost != server.Listener.Addr().String() {
		t.Errorf("server saw path %q, host %q", gotPath, gotHost)
	}
}

func TestTheServerMustBeAnHTTPSURL(t *testing.T) {
	for _, server := range []string{"http://host.ts.net", "host.ts.net", "https://", "https://a b"} {
		if _, err := newRemote(server, http.DefaultTransport); err == nil {
			t.Errorf("newRemote(%q) gave no error", server)
		}
	}
}

func TestConnectChoosesTheSocketOrTheServer(t *testing.T) {
	t.Setenv("NUDGE_SOCKET", "/tmp/nudge-test.sock")

	client, err := Connect("")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := client.Transport.(*http.Transport); !ok {
		t.Errorf("transport = %T, want the socket's", client.Transport)
	}

	if client, err = Connect("https://host.tailnet.ts.net:8443"); err != nil {
		t.Fatal(err)
	}

	if _, ok := client.Transport.(remote); !ok {
		t.Errorf("transport = %T, want the remote's", client.Transport)
	}

	if _, err := Connect("http://insecure"); err == nil {
		t.Error("an http URL gave no error")
	}
}
