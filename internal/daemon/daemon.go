package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aayush/torrcli/internal/engine"
	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/aayush/torrcli/internal/state"
	"github.com/aayush/torrcli/internal/transport"
)

const Version = "0.1.0-dev"

type Daemon struct {
	paths    platform.Paths
	started  time.Time
	config   model.Config
	session  model.Session
	torrents *torrentSession
	engine   engine.Engine
}

func New(paths platform.Paths, torrentEngine engine.Engine) *Daemon {
	return &Daemon{paths: paths, engine: torrentEngine}
}

func (d *Daemon) Run(ctx context.Context) error {
	defer d.engine.Close()
	if err := d.loadState(); err != nil {
		return err
	}
	d.torrents = newTorrentSession(ctx)

	lock, err := platform.AcquireLock(d.paths.LockFile)
	if err != nil {
		return err
	}
	defer lock.Close()

	listener, err := transport.Listen(d.paths.SocketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = transport.RemoveEndpoint(d.paths.SocketPath)
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

func (d *Daemon) handleRPC(ctx context.Context, request rpc.Request) (any, *rpc.Error) {
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
	case rpc.MethodTorrentAdd:
		var params rpc.AddTorrentParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, rpc.InvalidParams()
		}
		id, err := d.engine.Add(ctx, model.AddInput{Source: params.Source, SavePath: params.SavePath})
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		torrent, err := d.engine.Snapshot(ctx, id)
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		if err := d.torrents.put(ctx, torrent); err != nil {
			return nil, rpc.InternalError(err)
		}
		return rpc.AddTorrentResult{Torrent: torrent}, nil
	case rpc.MethodTorrentList:
		torrents, err := d.torrents.list(ctx)
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		return rpc.ListTorrentsResult{Torrents: torrents}, nil
	default:
		return nil, rpc.MethodNotFound(request.Method)
	}
}
