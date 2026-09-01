package daemon

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
)

func (d *Daemon) handleRPC(ctx context.Context, request rpc.Request) (any, *rpc.Error) {
	switch request.Method {
	case rpc.MethodDaemonPing:
		return rpc.PingResult{ProtocolVersion: rpc.Version}, nil
	case rpc.MethodDaemonInfo:
		return rpc.DaemonInfo{DaemonVersion: Version, ProtocolVersion: rpc.Version, StartedAt: d.started, ConfigFile: d.paths.ConfigFile, SessionFile: d.paths.SessionFile, SocketPath: d.paths.SocketPath}, nil
	case rpc.MethodTorrentAdd:
		d.controlMu.Lock()
		defer d.controlMu.Unlock()
		var p rpc.AddTorrentParams
		if json.Unmarshal(request.Params, &p) != nil {
			return nil, rpc.InvalidParams()
		}
		if p.SavePath == "" {
			p.SavePath = d.config.DownloadDirectory
		}
		id, created, err := d.engine.Add(ctx, model.AddInput{Source: p.Source, SavePath: p.SavePath})
		if err != nil {
			return nil, d.operationError("add", err)
		}
		t, err := d.engine.Snapshot(ctx, id)
		if err != nil {
			return nil, d.operationError("add", err)
		}
		if !created {
			return rpc.AddTorrentResult{Torrent: t}, nil
		}
		d.sessionMu.Lock()
		previous := cloneSession(d.session)
		d.sessionMu.Unlock()
		if err = d.recordTorrent(id, p, t); err != nil {
			_ = d.engine.Remove(context.WithoutCancel(ctx), id, false)
			return nil, d.operationError("add", err)
		}
		if err = d.torrents.put(context.WithoutCancel(ctx), t); err != nil {
			_ = d.replaceSession(context.WithoutCancel(ctx), previous)
			_ = d.engine.Remove(context.WithoutCancel(ctx), id, false)
			return nil, d.operationError("add", err)
		}
		return rpc.AddTorrentResult{Torrent: t}, nil
	case rpc.MethodTorrentList:
		ts, err := d.torrents.list(ctx)
		if err != nil {
			return nil, d.operationError("list", err)
		}
		return rpc.ListTorrentsResult{Torrents: ts}, nil
	case rpc.MethodTorrentGet:
		var p rpc.TorrentParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" {
			return nil, rpc.InvalidParams()
		}
		details, err := d.engine.Details(ctx, p.ID)
		if err != nil {
			return nil, d.operationError("get torrent details", err)
		}
		return rpc.TorrentDetailsResult{Details: details}, nil
	case rpc.MethodTorrentMove:
		var p rpc.MoveTorrentParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" || (p.Offset != -1 && p.Offset != 1) {
			return nil, rpc.InvalidParams()
		}
		torrents, err := d.moveTorrent(ctx, p.ID, p.Offset)
		if err != nil {
			return nil, d.operationError("move", err)
		}
		return rpc.ListTorrentsResult{Torrents: torrents}, nil
	case rpc.MethodTorrentPause, rpc.MethodTorrentResume:
		var p rpc.TorrentParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" {
			return nil, rpc.InvalidParams()
		}
		var t model.TorrentSnapshot
		var err error
		if request.Method == rpc.MethodTorrentPause {
			t, err = d.pauseTorrent(ctx, p.ID)
		} else {
			t, err = d.resumeTorrent(ctx, p.ID)
		}
		if err != nil {
			if request.Method == rpc.MethodTorrentPause {
				return nil, d.operationError("pause", err)
			}
			return nil, d.operationError("resume", err)
		}
		return rpc.TorrentResult{Torrent: t}, nil
	case rpc.MethodTorrentRemove:
		var p rpc.RemoveTorrentParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" {
			return nil, rpc.InvalidParams()
		}
		if err := d.removeTorrent(ctx, p); err != nil {
			return nil, d.operationError("remove", err)
		}
		return struct{}{}, nil
	case rpc.MethodTorrentSetFilePriority:
		var p rpc.SetFilePriorityParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" || p.FileIndex < 0 || !validFilePriority(p.Priority) {
			return nil, rpc.InvalidParams()
		}
		t, err := d.setFilePriority(ctx, p)
		if err != nil {
			return nil, d.operationError("priority", err)
		}
		return rpc.TorrentResult{Torrent: t}, nil
	default:
		return nil, rpc.MethodNotFound(request.Method)
	}
}

func (d *Daemon) operationError(operation string, err error) *rpc.Error {
	slog.Error("torrd operation failed", "operation", operation, "error", err)
	return rpc.OperationFailed(operation, err)
}
