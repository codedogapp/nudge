package phonedoor_test

import (
	"net/http"
	"net/url"
	"testing"
	"testing/synctest"

	"github.com/codedogapp/nudge/internal/inbox"
)

func TestCrossOriginAnswerIsRefused(t *testing.T) {
	tests := []struct {
		name, header, value string
	}{
		{"from another site", "Sec-Fetch-Site", "cross-site"},
		{"from another origin", "Origin", "https://evil.example"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				box, _ := asked(t, inbox.Question{Text: "Merge?", Options: []string{"Squash"}})

				req := form(t, answerPath(box), url.Values{"option": {"Squash"}})
				req.Header.Del("Sec-Fetch-Site")
				req.Header.Set(test.header, test.value)

				if rec := serve(box, req); rec.Code != http.StatusForbidden {
					t.Errorf("POST = %d, want 403", rec.Code)
				}

				if pending := box.Pending(); len(pending) != 1 {
					t.Errorf("pending = %+v, want the question to keep waiting", pending)
				}
			})
		})
	}
}
