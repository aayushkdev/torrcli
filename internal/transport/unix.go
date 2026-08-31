//go:build unix

package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func Listen(socketPath string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return nil, fmt.Errorf("create runtime directory: %w", err)
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove stale socket: %w", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("listen on daemon socket: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("secure daemon socket: %w", err)
	}
	return listener, nil
}

func DialContext(ctx context.Context, socketPath string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
}

func RemoveEndpoint(socketPath string) error {
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
