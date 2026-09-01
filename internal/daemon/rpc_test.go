package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/aayush/torrcli/internal/state"
)

func TestDaemonAddsAndListsTorrents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	torrentEngine := &recordingEngine{snapshot: model.TorrentSnapshot{
		ID:      "torrent-a",
		Name:    "example",
		AddedAt: addedAt,
	}}
	d := New(platform.Paths{SessionFile: filepath.Join(t.TempDir(), "session.json")}, nil)
	d.engine = torrentEngine
	d.session = model.DefaultSession()
	d.torrents = newTorrentSession(ctx)
	params, err := json.Marshal(rpc.AddTorrentParams{Source: "magnet:?xt=urn:btih:abc", SavePath: "/downloads"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	result, rpcError := d.handleRPC(ctx, rpc.Request{Method: rpc.MethodTorrentAdd, Params: params})
	if rpcError != nil {
		t.Fatalf("add RPC error: %#v", rpcError)
	}
	if torrentEngine.input.Source != "magnet:?xt=urn:btih:abc" || torrentEngine.input.SavePath != "/downloads" {
		t.Fatalf("engine input = %#v", torrentEngine.input)
	}
	addResult, ok := result.(rpc.AddTorrentResult)
	if !ok || addResult.Torrent != torrentEngine.snapshot {
		t.Fatalf("add result = %#v", result)
	}

	result, rpcError = d.handleRPC(ctx, rpc.Request{Method: rpc.MethodTorrentList})
	if rpcError != nil {
		t.Fatalf("list RPC error: %#v", rpcError)
	}
	listResult, ok := result.(rpc.ListTorrentsResult)
	if !ok || len(listResult.Torrents) != 1 || listResult.Torrents[0] != torrentEngine.snapshot {
		t.Fatalf("list result = %#v", result)
	}
	saved, err := state.LoadOrCreateSession(d.paths.SessionFile)
	if err != nil {
		t.Fatalf("load saved session: %v", err)
	}
	if len(saved.Order) != 1 || saved.Order[0] != "torrent-a" || saved.Torrents["torrent-a"].Source != torrentEngine.input.Source {
		t.Fatalf("saved session = %#v", saved)
	}
}

func TestDaemonRestoresSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addedAt := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	torrentEngine := &recordingEngine{snapshot: model.TorrentSnapshot{ID: "torrent-a", Name: "live"}}
	d := New(platform.Paths{}, nil)
	d.engine = torrentEngine
	d.session = model.Session{
		Order: []model.TorrentID{"torrent-a"},
		Torrents: map[model.TorrentID]model.TorrentRecord{
			"torrent-a": {
				Source:       "magnet:?xt=urn:btih:abc",
				Name:         "saved",
				SavePath:     "/downloads",
				DesiredState: model.TorrentStatePaused,
				AddedAt:      addedAt,
			},
		},
	}
	d.torrents = newTorrentSession(ctx)

	d.restoreSession(ctx)

	if torrentEngine.input.Source != "magnet:?xt=urn:btih:abc" || torrentEngine.input.SavePath != "/downloads" {
		t.Fatalf("engine input = %#v", torrentEngine.input)
	}
	if torrentEngine.pausedID != "torrent-a" {
		t.Fatalf("paused ID = %q", torrentEngine.pausedID)
	}
	torrents, err := d.torrents.list(ctx)
	if err != nil {
		t.Fatalf("list restored torrents: %v", err)
	}
	if len(torrents) != 1 || torrents[0].AddedAt != addedAt || torrents[0].Name != "live" {
		t.Fatalf("restored torrents = %#v", torrents)
	}
}

type recordingEngine struct {
	input    model.AddInput
	snapshot model.TorrentSnapshot
	pausedID model.TorrentID
}

func (e *recordingEngine) Add(_ context.Context, input model.AddInput) (model.TorrentID, error) {
	e.input = input
	return e.snapshot.ID, nil
}

func (e *recordingEngine) Pause(_ context.Context, id model.TorrentID) error {
	e.pausedID = id
	return nil
}

func (e *recordingEngine) Resume(context.Context, model.TorrentID) error { return nil }

func (e *recordingEngine) Remove(context.Context, model.TorrentID, bool) error { return nil }

func (e *recordingEngine) SetFilePriority(context.Context, model.TorrentID, int, model.FilePriority) error {
	return nil
}

func (e *recordingEngine) Snapshot(context.Context, model.TorrentID) (model.TorrentSnapshot, error) {
	return e.snapshot, nil
}

func (e *recordingEngine) Events() <-chan model.EngineEvent { return nil }

func (e *recordingEngine) Close() error { return nil }
