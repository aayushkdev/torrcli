package anacrolix

import (
	"bytes"
	"context"
	"crypto/sha1"
	"os"
	"path/filepath"
	"testing"

	"github.com/aayush/torrcli/internal/model"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func TestSequentialWindowAdvancesAndRespectsSkippedFiles(t *testing.T) {
	directory := t.TempDir()
	config := torrent.NewDefaultClientConfig()
	config.DisableTCP, config.DisableUTP = true, true
	config.NoDHT, config.DisableTrackers, config.NoDefaultPortForwarding = true, true, true
	config.DisableWebtorrent = true
	store := newFileStorage(directory)
	config.DefaultStorage = store
	client, err := torrent.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	defer store.Close()
	const pieceSize = 1 << 20
	hash := sha1.Sum(make([]byte, pieceSize))
	info := metainfo.Info{Name: "movie.mkv", PieceLength: pieceSize, Length: 20 * pieceSize}
	info.Pieces = append(info.Pieces, hash[:]...)
	hash = sha1.Sum(bytes.Repeat([]byte{1}, pieceSize))
	for range 19 {
		info.Pieces = append(info.Pieces, hash[:]...)
	}
	encoded, err := bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "movie.mkv"), bytes.Repeat([]byte{2}, int(info.Length)), 0600); err != nil {
		t.Fatal(err)
	}
	tor, err := client.AddTorrent(&metainfo.MetaInfo{InfoBytes: encoded})
	if err != nil {
		t.Fatal(err)
	}
	tor.Files()[0].SetPriority(torrent.PiecePriorityNormal)
	id := model.TorrentID(tor.InfoHash().HexString())
	engine := &Engine{torrents: map[model.TorrentID]torrentEntry{id: {torrent: tor}}}
	if err := engine.SetSequential(context.Background(), id, true); err != nil {
		t.Fatal(err)
	}
	entry := engine.torrents[id]
	if len(entry.boosted) != 16 || entry.boosted[0] != torrent.PiecePriorityNow {
		t.Fatalf("initial window: %v", entry.boosted)
	}
	file, err := os.OpenFile(filepath.Join(directory, "movie.mkv"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteAt(make([]byte, pieceSize), 0)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := tor.VerifyDataContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Snapshot(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	entry = engine.torrents[id]
	if _, exists := entry.boosted[0]; exists || entry.boosted[1] != torrent.PiecePriorityNow || entry.boosted[16] != torrent.PiecePriorityReadahead {
		t.Fatalf("advanced window: %v", entry.boosted)
	}
	if err := engine.SetFilePriority(context.Background(), id, 0, model.FilePrioritySkip); err != nil {
		t.Fatal(err)
	}
	if len(engine.torrents[id].boosted) != 0 || tor.Piece(1).State().Priority != torrent.PiecePriorityNone {
		t.Fatal("skipped file still requested")
	}
	if err := engine.SetFilePriority(context.Background(), id, 0, model.FilePriorityHigh); err != nil {
		t.Fatal(err)
	}
	if err := engine.SetSequential(context.Background(), id, false); err != nil {
		t.Fatal(err)
	}
	if len(engine.torrents[id].boosted) != 0 || tor.Piece(1).State().Priority != torrent.PiecePriorityHigh {
		t.Fatal("disabling sequential did not restore file priority")
	}
}
