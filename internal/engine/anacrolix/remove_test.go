package anacrolix

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aayush/torrcli/internal/model"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

func TestRemoveDataKeepsUnrelatedFiles(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "bundle", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bundle/movie", "bundle/sub/subtitle", "bundle/keep.txt", "other"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	info := &metainfo.Info{Name: "bundle", Files: []metainfo.FileInfo{{Path: []string{"movie"}, Length: 4}, {Path: []string{"sub", "subtitle"}, Length: 4}, {Path: []string{"missing"}, Length: 4}}}
	root, paths, err := prepareDataRemoval(directory, info)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := removeDataFiles(root, paths); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bundle/movie", "bundle/sub"} {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("%s was not removed: %v", name, err)
		}
	}
	for _, name := range []string{"bundle/keep.txt", "other"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || string(data) != "data" {
			t.Fatalf("unrelated %s changed", name)
		}
	}
}

func TestRemoveDataRejectsTraversalSymlinksAndDirectories(t *testing.T) {
	for _, kind := range []string{"traversal", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			outside := filepath.Join(t.TempDir(), "keep")
			if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			name := "movie"
			switch kind {
			case "traversal":
				name = "../keep"
			case "symlink":
				if err := os.Symlink(outside, filepath.Join(directory, name)); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(filepath.Join(directory, name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			root, _, err := prepareDataRemoval(directory, &metainfo.Info{Name: name, Length: 4})
			if root != nil {
				root.Close()
			}
			if err == nil {
				t.Fatal("unsafe path accepted")
			}
			data, err := os.ReadFile(outside)
			if err != nil || string(data) != "keep" {
				t.Fatal("outside file changed")
			}
		})
	}
}

func TestEngineRemoveWithAndWithoutData(t *testing.T) {
	for _, deleteData := range []bool{false, true} {
		t.Run(map[bool]string{false: "keep", true: "delete"}[deleteData], func(t *testing.T) {
			directory := t.TempDir()
			cfg := torrent.NewDefaultClientConfig()
			cfg.DisableTCP, cfg.DisableUTP, cfg.NoDHT, cfg.DisableTrackers, cfg.NoDefaultPortForwarding, cfg.DisableWebtorrent = true, true, true, true, true, true
			cfg.DefaultStorage = storage.NewFileOpts(storage.NewFileClientOpts{PieceCompletion: storage.NewMapPieceCompletion()})
			client, err := torrent.NewClient(cfg)
			if err != nil {
				t.Fatal(err)
			}
			e := &Engine{client: client, torrents: make(map[model.TorrentID]torrentEntry), events: make(chan model.EngineEvent)}
			defer e.Close()
			info := metainfo.Info{Name: "movie", Length: 4, PieceLength: 4, Pieces: make([]byte, 20)}
			encoded, err := bencode.Marshal(info)
			if err != nil {
				t.Fatal(err)
			}
			meta := metainfo.MetaInfo{InfoBytes: encoded}
			source := filepath.Join(t.TempDir(), "movie.torrent")
			file, err := os.Create(source)
			if err != nil {
				t.Fatal(err)
			}
			err = meta.Write(file)
			file.Close()
			if err != nil {
				t.Fatal(err)
			}
			id, _, err := e.Add(context.Background(), model.AddInput{Source: source, SavePath: directory, Paused: true})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "movie")
			if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := e.Remove(context.Background(), id, deleteData); err != nil {
				t.Fatal(err)
			}
			_, err = os.Stat(path)
			if deleteData && !os.IsNotExist(err) {
				t.Fatalf("file not deleted: %v", err)
			}
			if !deleteData && err != nil {
				t.Fatalf("file not kept: %v", err)
			}
			if _, err := e.Snapshot(context.Background(), id); err == nil {
				t.Fatal("torrent still registered")
			}
		})
	}
}

func TestRemoveBeforeMetadataWithDeleteData(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "movie")
	if err := os.WriteFile(path, []byte("unrelated"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := torrent.NewDefaultClientConfig()
	cfg.DisableTCP, cfg.DisableUTP, cfg.NoDHT, cfg.DisableTrackers, cfg.NoDefaultPortForwarding, cfg.DisableWebtorrent = true, true, true, true, true, true
	cfg.DefaultStorage = storage.NewFileOpts(storage.NewFileClientOpts{PieceCompletion: storage.NewMapPieceCompletion()})
	client, err := torrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{client: client, torrents: make(map[model.TorrentID]torrentEntry), events: make(chan model.EngineEvent)}
	defer e.Close()
	id, _, err := e.Add(context.Background(), model.AddInput{Source: "magnet:?xt=urn:btih:0123456789012345678901234567890123456789&dn=movie", SavePath: directory})
	if err != nil {
		t.Fatal(err)
	}
	if e.torrents[id].torrent.Info() != nil {
		t.Fatal("expected unavailable metadata")
	}
	if err := e.Remove(context.Background(), id, true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Snapshot(context.Background(), id); err == nil {
		t.Fatal("torrent still registered")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "unrelated" {
		t.Fatal("removal changed an unrelated file")
	}
}
