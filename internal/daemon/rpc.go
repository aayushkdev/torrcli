package daemon

import (
	"context"
	"encoding/json"

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
			return nil, rpc.InternalError(err)
		}
		t, err := d.engine.Snapshot(ctx, id)
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		if !created {
			return rpc.AddTorrentResult{Torrent: t}, nil
		}
		if err = d.recordTorrent(id, p, t); err != nil {
			_ = d.engine.Remove(context.WithoutCancel(ctx), id, false)
			return nil, rpc.InternalError(err)
		}
		if err = d.torrents.put(ctx, t); err != nil {
			return nil, rpc.InternalError(err)
		}
		return rpc.AddTorrentResult{Torrent: t}, nil
	case rpc.MethodTorrentList:
		ts, err := d.torrents.list(ctx)
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		return rpc.ListTorrentsResult{Torrents: ts}, nil
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
			return nil, rpc.InternalError(err)
		}
		return rpc.TorrentResult{Torrent: t}, nil
	case rpc.MethodTorrentRemove:
		var p rpc.RemoveTorrentParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" {
			return nil, rpc.InvalidParams()
		}
		if err := d.removeTorrent(ctx, p); err != nil {
			return nil, rpc.InternalError(err)
		}
		return struct{}{}, nil
	case rpc.MethodTorrentSetFilePriority:
		var p rpc.SetFilePriorityParams
		if json.Unmarshal(request.Params, &p) != nil || p.ID == "" || p.FileIndex < 0 || !validFilePriority(p.Priority) {
			return nil, rpc.InvalidParams()
		}
		t, err := d.setFilePriority(ctx, p)
		if err != nil {
			return nil, rpc.InternalError(err)
		}
		return rpc.TorrentResult{Torrent: t}, nil
	default:
		return nil, rpc.MethodNotFound(request.Method)
	}
}
