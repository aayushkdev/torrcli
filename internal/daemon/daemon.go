package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
	paths     platform.Paths
	newEngine func() (engine.Engine, error)
	started   time.Time
	config    model.Config
	session   model.Session
	sessionMu sync.Mutex
	controlMu sync.Mutex
	torrents  *torrentSession
	engine    engine.Engine
}

func New(paths platform.Paths, newEngine func() (engine.Engine, error)) *Daemon {
	return &Daemon{paths: paths, newEngine: newEngine}
}

func (d *Daemon) Run(ctx context.Context) error {
	if err := d.loadState(); err != nil {
		return err
	}

	lock, err := platform.AcquireLock(d.paths.LockFile)
	if err != nil {
		return err
	}
	defer lock.Close()

	d.engine, err = d.newEngine()
	if err != nil {
		return err
	}
	defer d.engine.Close()
	d.torrents = newTorrentSession(ctx)
	d.restoreSession(ctx)
	go d.refreshTorrents(ctx)

	listener, err := transport.Listen(d.paths.SocketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = transport.RemoveEndpoint(d.paths.SocketPath)
	}()

	d.started = time.Now().UTC()
	slog.Info("torrd started", "socket", d.paths.SocketPath)
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
		slog.Info("torrd stopping")
		_ = listener.Close()
		closeConnections()
		connections.Wait()
		return nil
	case err := <-acceptErrors:
		if err != nil {
			slog.Error("torrd listener failed", "error", err)
		}
		closeConnections()
		connections.Wait()
		return err
	}
}

func (d *Daemon) refreshTorrents(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			torrents, err := d.torrents.list(ctx)
			if err != nil {
				continue
			}
			for _, torrent := range torrents {
				if updated, err := d.engine.Snapshot(ctx, torrent.ID); err == nil {
					_ = d.torrents.put(ctx, updated)
				}
			}
		}
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

func (d *Daemon) restoreSession(ctx context.Context) {
	for _, id := range d.session.Order {
		record := d.session.Torrents[id]
		torrent := model.TorrentSnapshot{
			ID:      id,
			Name:    record.Name,
			State:   model.TorrentStateError,
			AddedAt: record.AddedAt,
		}
		restoredID, _, err := d.engine.Add(ctx, model.AddInput{Source: record.Source, SavePath: record.SavePath, FilePriorities: filePriorities(record.FilePriorities)})
		if err == nil && restoredID != id {
			err = fmt.Errorf("restored torrent ID %q does not match session ID %q", restoredID, id)
		}
		if err == nil && record.DesiredState == model.TorrentStatePaused {
			err = d.engine.Pause(ctx, id)
		}
		if err == nil {
			snapshot, snapshotErr := d.engine.Snapshot(ctx, id)
			if snapshotErr != nil {
				err = snapshotErr
			} else {
				torrent = snapshot
				torrent.AddedAt = record.AddedAt
			}
		}
		if err != nil {
			torrent.Error = err.Error()
		}
		_ = d.torrents.put(context.WithoutCancel(ctx), torrent)
	}
}

func filePriorities(priorities map[string]model.FilePriority) map[int]model.FilePriority {
	result := make(map[int]model.FilePriority, len(priorities))
	for index, priority := range priorities {
		if parsed, err := strconv.Atoi(index); err == nil && parsed >= 0 {
			result[parsed] = priority
		}
	}
	return result
}

func (d *Daemon) recordTorrent(id model.TorrentID, params rpc.AddTorrentParams, torrent model.TorrentSnapshot) error {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()

	session := cloneSession(d.session)
	record, exists := session.Torrents[id]
	if !exists {
		session.Order = append(session.Order, id)
	}
	record.Source = params.Source
	record.Name = torrent.Name
	record.SavePath = params.SavePath
	record.DesiredState = model.TorrentStateDownloading
	record.AddedAt = torrent.AddedAt
	session.Torrents[id] = record
	if err := state.SaveSession(d.paths.SessionFile, session); err != nil {
		return err
	}
	d.session = session
	return nil
}

func cloneSession(session model.Session) model.Session {
	clone := model.Session{
		Order:    append([]model.TorrentID(nil), session.Order...),
		Torrents: make(map[model.TorrentID]model.TorrentRecord, len(session.Torrents)),
	}
	for id, record := range session.Torrents {
		clone.Torrents[id] = record
	}
	return clone
}
