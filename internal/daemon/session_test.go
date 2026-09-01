package daemon

import (
	"context"
	"testing"

	"github.com/aayush/torrcli/internal/model"
)

func TestTorrentSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := newTorrentSession(ctx)
	torrent := model.TorrentSnapshot{ID: "torrent-a", Name: "example"}
	if err := session.put(ctx, torrent); err != nil {
		t.Fatalf("put torrent: %v", err)
	}

	actual, err := session.get(ctx, torrent.ID)
	if err != nil {
		t.Fatalf("get torrent: %v", err)
	}
	if actual != torrent {
		t.Fatalf("torrent = %#v, want %#v", actual, torrent)
	}
}
