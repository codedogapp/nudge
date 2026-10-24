package phonedoor_test

import (
	"encoding/json/v2"
	"fmt"
	"image/png"
	"net/http"
	"slices"
	"testing"

	"github.com/codedogapp/nudge/internal/inbox"
)

type icon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type"`
}

// served decodes the manifest that the door serves into target, a pointer.
func served(t *testing.T, target any) {
	t.Helper()

	var box inbox.Inbox

	body := get(t, &box, http.MethodGet, "/manifest.json").Body.Bytes()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("manifest: %v", err)
	}
}

func TestManifestDescribesTheApp(t *testing.T) {
	type app struct {
		Name            string `json:"name"`
		ShortName       string `json:"short_name"`
		StartURL        string `json:"start_url"`
		Display         string `json:"display"`
		ThemeColor      string `json:"theme_color"`
		BackgroundColor string `json:"background_color"`
	}

	var got app
	served(t, &got)

	want := app{
		Name:            "nudge",
		ShortName:       "nudge",
		StartURL:        "/",
		Display:         "standalone",
		ThemeColor:      "#fafaf8",
		BackgroundColor: "#fafaf8",
	}
	if got != want {
		t.Errorf("manifest = %+v, want %+v", got, want)
	}
}

func TestIconsAreServedAtTheirSize(t *testing.T) {
	var got struct {
		Icons []icon `json:"icons"`
	}
	served(t, &got)

	sizes := []string{"192x192", "512x512"}
	if !slices.EqualFunc(
		got.Icons,
		sizes,
		func(i icon, size string) bool { return i.Sizes == size },
	) {
		t.Errorf("manifest icons = %+v, want one of each size in %v", got.Icons, sizes)
	}

	appleTouch := icon{Src: "/apple-touch-icon.png", Sizes: "180x180", Type: "image/png"}
	for _, want := range append(got.Icons, appleTouch) {
		var box inbox.Inbox

		rec := get(t, &box, http.MethodGet, want.Src)
		if typ := rec.Header().Get("Content-Type"); rec.Code != http.StatusOK || typ != want.Type {
			t.Errorf("GET %s = %d as %q, want 200 as %q", want.Src, rec.Code, typ, want.Type)
		}

		config, err := png.DecodeConfig(rec.Body)
		if err != nil {
			t.Errorf("GET %s: %v", want.Src, err)

			continue
		}

		if size := fmt.Sprintf("%dx%d", config.Width, config.Height); size != want.Sizes {
			t.Errorf("GET %s = a %s PNG, want %s", want.Src, size, want.Sizes)
		}
	}
}
