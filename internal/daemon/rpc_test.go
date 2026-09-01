package daemon

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/rpc"
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
	d := New(platform.Paths{}, torrentEngine)
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
}

type recordingEngine struct {
	input    model.AddInput
	snapshot model.TorrentSnapshot
}

func (e *recordingEngine) Add(_ context.Context, input model.AddInput) (model.TorrentID, error) {
	e.input = input
	return e.snapshot.ID, nil
}

func (e *recordingEngine) Pause(context.Context, model.TorrentID) error { return nil }

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
