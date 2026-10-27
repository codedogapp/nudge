package setup

import (
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

// WriteSettings writes the settings file at path with the server's address, an https URL, and
// returns a line that says what it did. A file that exists stays as it is.
func WriteSettings(path, server string) (string, error) {
	if parsed, err := url.Parse(
		server,
	); err != nil || parsed.Scheme != "https" ||
		parsed.Host == "" {
		return "", fmt.Errorf("the server %q isn't an https URL", server)
	}

	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return "", fmt.Errorf("make the settings folder: %w", err)
	}

	//nolint:gosec // G304: the path is nudge's own settings file, in the user's config folder.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if errors.Is(err, fs.ErrExist) {
		return "settings: kept " + path, nil
	}

	if err != nil {
		return "", fmt.Errorf("write the settings file: %w", err)
	}

	// %q's escapes are JSON's, for a URL that url.Parse took.
	_, err = fmt.Fprintf(file, "{\n  \"server\": %q\n}\n", server)

	return "settings: wrote " + path, errors.Join(err, file.Close())
}
