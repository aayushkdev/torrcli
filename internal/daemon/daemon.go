// Package daemon contains the long-running torrd service.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/aayush/torrcli/internal/state"
)

const Version = "0.1.0-dev"

// Daemon owns the persistent state and local RPC service.
type Daemon struct {
	paths   platform.Paths
	started time.Time
	config  model.Config
	session model.Session
}

// New creates a daemon that uses paths for its local state and endpoint.
func New(paths platform.Paths) *Daemon {
	return &Daemon{paths: paths}
}

// Run starts the local socket service and blocks until ctx is canceled.
func (d *Daemon) Run(ctx context.Context) error {
	if err := d.loadState(); err != nil {
		return err
	}

	lock, err := platform.AcquireLock(d.paths.LockFile)
	if err != nil {
		return err
	}
	defer lock.Close()

	listener, err := d.listen()
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(d.paths.SocketPath)
	}()

	d.started = time.Now().UTC()
	var connections sync.WaitGroup
	var connectionMu sync.Mutex
	activeConnections := make(map[net.Conn]struct{})
	closeConnections := func() {
		connectionMu.Lock()
		defer connectionMu.Unlock()
		for connection := range activeConnections {
			_ = connection.Close()
		}
	}
	acceptErrors := make(chan error, 1)
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				if errors.Is(acceptErr, net.ErrClosed) {
					acceptErrors <- nil
					return
				}
				acceptErrors <- acceptErr
				return
			}
			connections.Add(1)
			connectionMu.Lock()
			activeConnections[connection] = struct{}{}
			connectionMu.Unlock()
			go func() {
				defer connections.Done()
				defer func() {
					connectionMu.Lock()
					delete(activeConnections, connection)
					connectionMu.Unlock()
				}()
				_ = rpc.ServeConn(ctx, connection, d.handleRPC)
			}()
		}
	}()

	select {
	case <-ctx.Done():
		_ = listener.Close()
		closeConnections()
		connections.Wait()
		return nil
	case err := <-acceptErrors:
		closeConnections()
		connections.Wait()
		return err
	}
}

func (d *Daemon) loadState() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}

	d.config, err = state.LoadOrCreateConfig(d.paths.ConfigFile, filepath.Join(home, "Downloads"))
	if err != nil {
		return err
	}
	d.session, err = state.LoadOrCreateSession(d.paths.SessionFile)
	if err != nil {
		return err
	}
	return nil
}

func (d *Daemon) listen() (*net.UnixListener, error) {
	if err := os.MkdirAll(d.paths.RuntimeDir, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime directory: %w", err)
	}
	if err := os.Remove(d.paths.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove stale socket: %w", err)
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: d.paths.SocketPath, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("listen on daemon socket: %w", err)
	}
	if err := os.Chmod(d.paths.SocketPath, 0o600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("secure daemon socket: %w", err)
	}
	return listener, nil
}

func (d *Daemon) handleRPC(_ context.Context, request rpc.Request) (any, *rpc.Error) {
	switch request.Method {
	case rpc.MethodDaemonPing:
		return rpc.PingResult{ProtocolVersion: rpc.Version}, nil
	case rpc.MethodDaemonInfo:
		return rpc.DaemonInfo{
			DaemonVersion:   Version,
			ProtocolVersion: rpc.Version,
			StartedAt:       d.started,
			ConfigFile:      d.paths.ConfigFile,
			SessionFile:     d.paths.SessionFile,
			SocketPath:      d.paths.SocketPath,
		}, nil
	default:
		return nil, rpc.MethodNotFound(request.Method)
	}
}
