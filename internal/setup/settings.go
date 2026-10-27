package setup

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
)

// Folder and file modes: only the user can open them.
const (
	folderMode = 0o700
	fileMode   = 0o600
)

type settings struct {
	Server string `json:"server"`
}

// WriteSettings writes the settings file at path with the server's address, an https URL, and
// returns a line that says what it did. A file that exists stays as it is.
func WriteSettings(path, server string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return "settings: kept " + path, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("check the settings file: %w", err)
	}

	if parsed, err := url.Parse(
		server,
	); err != nil || parsed.Scheme != "https" ||
		parsed.Host == "" {
		return "", fmt.Errorf("the server %q isn't an https URL", server)
	}

	data, err := json.Marshal(settings{Server: server}, jsontext.WithIndent("  "))
	if err != nil {
		return "", fmt.Errorf("encode the settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return "", fmt.Errorf("make the settings folder: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), fileMode); err != nil {
		return "", fmt.Errorf("write the settings file: %w", err)
	}

	return "settings: wrote " + path, nil
}
