package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// fileSettings are the settings that the settings file gives. A setting it leaves out is empty.
type fileSettings struct {
	Login         string `json:"login"`
	Host          string `json:"host"`
	Listen        string `json:"listen"`
	Subscriptions string `json:"subscriptions"`
}

// settingsFile is the settings file's path: nudge/settings.json in the user's config folder,
// next to the agent socket and the subscriptions.
func settingsFile(config string) string {
	return filepath.Join(config, "nudge", "settings.json")
}

// readSettingsFile reads the settings file at path. A missing file gives no settings.
func readSettingsFile(path string) (fileSettings, error) {
	//nolint:gosec // G304: the path is nudge's own settings file, in the user's config folder.
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fileSettings{}, nil
	}

	if err != nil {
		return fileSettings{}, fmt.Errorf("read the settings file: %w", err)
	}

	var file fileSettings
	if err := json.Unmarshal(data, &file, json.RejectUnknownMembers(true)); err != nil {
		return fileSettings{}, invalidSettings(path, err)
	}

	// launchd starts nudge serve in /, and JSON doesn't expand ~.
	if file.Subscriptions != "" && !filepath.IsAbs(file.Subscriptions) {
		return fileSettings{}, invalidSettings(path,
			fmt.Errorf("subscriptions %q isn't an absolute path", file.Subscriptions))
	}

	return file, nil
}

func invalidSettings(path string, err error) error {
	return fmt.Errorf("the settings file %s is invalid: %w", path, err)
}
