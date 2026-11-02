package anacrolix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

type fileStorage struct {
	storage.ClientImplCloser
	directory string
}

func newFileStorage(directory string) storage.ClientImplCloser {
	return &fileStorage{
		ClientImplCloser: storage.NewFileOpts(storage.NewFileClientOpts{ClientBaseDir: directory, UsePartFiles: g.Some(false)}),
		directory:        directory,
	}
}

func (s *fileStorage) OpenTorrent(ctx context.Context, info *metainfo.Info, hash metainfo.Hash) (storage.TorrentImpl, error) {
	if err := migratePartFiles(s.directory, info); err != nil {
		return storage.TorrentImpl{}, err
	}
	return s.ClientImplCloser.OpenTorrent(ctx, info, hash)
}

func migratePartFiles(directory string, info *metainfo.Info) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	paths := make(map[string]bool)
	for _, file := range info.UpvertedFiles() {
		parts := []string{}
		if info.BestName() != metainfo.NoName {
			parts = append(parts, info.BestName())
		}
		parts = append(parts, file.BestPath()...)
		path := filepath.Join(parts...)
		if !filepath.IsLocal(path) {
			return fmt.Errorf("invalid torrent file path %q", path)
		}
		paths[path] = true
	}
	for path := range paths {
		part := path + ".part"
		// A real torrent filename ending in .part must never be treated as legacy data.
		if paths[part] {
			continue
		}
		stat, err := root.Lstat(part)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !stat.Mode().IsRegular() {
			return fmt.Errorf("partial file %q is not a regular file", part)
		}
		// Linking refuses to overwrite an existing destination and confines both paths
		// to the download root. Only remove the old name after preserving its data.
		if err := root.Link(part, path); err != nil {
			target, statErr := root.Lstat(path)
			if statErr != nil || !os.SameFile(stat, target) {
				return fmt.Errorf("preserve partial download %q: %w", part, err)
			}
		}
		if err := root.Remove(part); err != nil {
			return err
		}
	}
	return nil
}
