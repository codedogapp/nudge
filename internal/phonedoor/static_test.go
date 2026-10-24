package phonedoor_test

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestDoorServesItsFiles(t *testing.T) {
	tests := []struct {
		name, contentType string
	}{
		{"style.css", "text/css; charset=utf-8"},
		{"grain.svg", "image/svg+xml"},
		{"icon.svg", "image/svg+xml"},
		{"apple-touch-icon.png", "image/png"},
		{"icon-192.png", "image/png"},
		{"icon-512.png", "image/png"},
		{"manifest.json", "application/json"},
	}

	for _, test := range tests {
		want, err := os.ReadFile("static/" + test.name)
		if err != nil {
			t.Fatal(err)
		}

		var box inbox.Inbox

		rec := get(t, &box, http.MethodGet, "/"+test.name)
		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), want) {
			t.Errorf("GET /%s = %d with %d bytes, want 200 with the %d bytes of static/%[1]s",
				test.name, rec.Code, rec.Body.Len(), len(want))
		}

		for header, value := range map[string]string{
			"Content-Type":  test.contentType,
			"Cache-Control": "no-cache",
		} {
			if got := rec.Header().Get(header); got != value {
				t.Errorf("GET /%s: %s = %q, want %q", test.name, header, got, value)
			}
		}
	}
}

func TestDoorServesNoOtherFile(t *testing.T) {
	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodHead, "/style.css", http.StatusOK},
		{http.MethodPost, "/style.css", http.StatusMethodNotAllowed},
		{http.MethodPost, "/manifest.json", http.StatusMethodNotAllowed},
		{http.MethodGet, "/style.css/", http.StatusNotFound},
		{http.MethodGet, "/static/style.css", http.StatusNotFound},
		{http.MethodGet, "/static/", http.StatusNotFound},
		{http.MethodGet, "/favicon.ico", http.StatusNotFound},
		{http.MethodGet, "/page.html", http.StatusNotFound},
		{http.MethodGet, "/static.go", http.StatusNotFound},
	}

	for _, test := range tests {
		var box inbox.Inbox

		if got := get(t, &box, test.method, test.path).Code; got != test.want {
			t.Errorf("%s %s = %d, want %d", test.method, test.path, got, test.want)
		}
	}
}

func TestPageLinksItsFiles(t *testing.T) {
	var box inbox.Inbox

	body := get(t, &box, http.MethodGet, "/").Body.String()
	for _, want := range []string{
		`<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">`,
		`<meta name="theme-color" content="#fafaf8" media="(prefers-color-scheme: light)">`,
		`<meta name="theme-color" content="#131312" media="(prefers-color-scheme: dark)">`,
		`<link rel="icon" href="/icon.svg" type="image/svg+xml">`,
		`<link rel="apple-touch-icon" href="/apple-touch-icon.png" sizes="180x180">`,
		`<link rel="manifest" href="/manifest.json">`,
		`<link rel="stylesheet" href="/style.css">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page = %s, want it to contain %s", body, want)
		}
	}
}
