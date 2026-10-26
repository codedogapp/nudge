package setup

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Folder and file modes: only the user can open them, as nudge serve needs.
const (
	folderMode = 0o700
	fileMode   = 0o600
)

type tailscaleStatus struct {
	Self struct {
		UserID  int64
		DNSName string
	}
	User map[string]struct{ LoginName string }
}

type settings struct {
	Login string `json:"login"`
	Host  string `json:"host"`
}

// WriteSettings writes the settings file at path from what `tailscale status --json` says about
// this Mac, and returns a line that says what it did. A file that exists stays as it is.
func WriteSettings(ctx context.Context, run Run, path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return "settings: kept " + path, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("check the settings file: %w", err)
	}

	out, err := run(ctx, "tailscale", "status", "--json")
	if err != nil {
		return "", fmt.Errorf("read the tailnet's status: %w", err)
	}

	found, err := parseStatus(out)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(found, jsontext.WithIndent("  "))
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

func parseStatus(out []byte) (settings, error) {
	var status tailscaleStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return settings{}, fmt.Errorf("read the tailnet's status: %w", err)
	}

	login := status.User[strconv.FormatInt(status.Self.UserID, 10)].LoginName
	host := strings.TrimSuffix(status.Self.DNSName, ".")

	if login == "" || host == "" {
		return settings{}, errors.New("the tailnet's status has no login or host: is Tailscale up?")
	}

	return settings{Login: login, Host: host}, nil
}
