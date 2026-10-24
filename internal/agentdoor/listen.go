package agentdoor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

var (
	// ErrOpenFolder means other users can open the socket's folder.
	ErrOpenFolder = errors.New("other users can open the socket's folder")
	// ErrNotSocket means something other than a socket is at the socket's path.
	ErrNotSocket = errors.New("the socket's path holds something else")
	// ErrRunning means a server already answers on the socket.
	ErrRunning = errors.New("nudge serve is already running")
)

const (
	// privateFolder lets only the user open the socket's folder.
	privateFolder fs.FileMode = 0o700
	// othersBits are the folder permissions of the group and of the other users.
	othersBits fs.FileMode = 0o077
)

// Listen listens on the agent socket at path, in a folder that only the user can open. It makes
// the folder if it's missing; it refuses one that others can open rather than change its mode.
func Listen(ctx context.Context, path string) (net.Listener, error) {
	folder := filepath.Dir(path)
	if err := os.MkdirAll(folder, privateFolder); err != nil {
		return nil, fmt.Errorf("make the socket's folder: %w", err)
	}

	info, err := os.Stat(folder)
	if err != nil {
		return nil, fmt.Errorf("check the socket's folder: %w", err)
	}

	if info.Mode().Perm()&othersBits != 0 {
		return nil, fmt.Errorf("%w: %s has mode %#o", ErrOpenFolder, folder, info.Mode().Perm())
	}

	if err := removeStale(ctx, path); err != nil {
		return nil, err
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on the agent socket: %w", err)
	}

	return listener, nil
}

// removeStale removes the socket that a server which has gone left behind.
func removeStale(ctx context.Context, path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("check the agent socket: %w", err)
	}

	if info.Mode().Type() != fs.ModeSocket {
		return fmt.Errorf("%w: %s", ErrNotSocket, path)
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err == nil {
		_ = conn.Close()
		return ErrRunning
	}

	// Only a refused dial proves the server has gone; another error leaves its socket alone.
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Errorf("dial the agent socket: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove the old agent socket: %w", err)
	}

	return nil
}
