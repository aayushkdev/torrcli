package anacrolix

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anacrolix/torrent/metainfo"
)

func TestStoragePreservesPartialDownloadWithoutSuffix(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "movie.mkv")
	if err := os.WriteFile(path+".part", []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	info := &metainfo.Info{Name: "movie.mkv", Length: 8, PieceLength: 8, Pieces: make([]byte, 20)}
	store := newFileStorage(directory)
	defer store.Close()
	tor, err := store.OpenTorrent(context.Background(), info, metainfo.Hash{})
	if err != nil {
		t.Fatal(err)
	}
	defer tor.Close()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "existing" {
		t.Fatalf("data = %q, error = %v", data, err)
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Fatalf("legacy name still exists: %v", err)
	}
	if _, err := tor.Piece(info.Piece(0)).WriteAt([]byte("new data"), 0); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "new data" {
		t.Fatalf("updated data = %q, error = %v", data, err)
	}
}

func TestPartMigrationRefusesConflictsAndEscapingSymlinks(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "conflict", true: "symlink"}[symlink], func(t *testing.T) {
			directory := t.TempDir()
			other := filepath.Join(t.TempDir(), "untouched")
			if err := os.WriteFile(other, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "movie")
			if symlink {
				if err := os.Symlink(other, path+".part"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path+".part", []byte("partial"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := migratePartFiles(directory, &metainfo.Info{Name: "movie", Length: 1}); err == nil {
				t.Fatal("expected refusal")
			}
			preserved := other
			if !symlink {
				preserved = path
			}
			data, err := os.ReadFile(preserved)
			if err != nil || string(data) != "original" {
				t.Fatalf("destination modified: %q, %v", data, err)
			}
			if _, err := os.Lstat(path + ".part"); err != nil {
				t.Fatalf("partial data lost: %v", err)
			}
		})
	}
}
