package anacrolix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anacrolix/torrent/metainfo"
)

func torrentFilePaths(info *metainfo.Info) ([]string, error) {
	paths := []string{}
	for _, file := range info.UpvertedFiles() {
		parts := []string{}
		if info.BestName() != metainfo.NoName {
			parts = append(parts, info.BestName())
		}
		path := filepath.Join(append(parts, file.BestPath()...)...)
		if !filepath.IsLocal(path) || path == "." {
			return nil, fmt.Errorf("invalid torrent file path %q", path)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func prepareDataRemoval(directory string, info *metainfo.Info) (*os.Root, []string, error) {
	paths, err := torrentFilePaths(info)
	if err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	for _, path := range paths {
		// Refuse symlink components, including links within the download directory:
		// deleting through them could target a different torrent's files.
		parts := strings.Split(path, string(filepath.Separator))
		for i := range parts {
			current := filepath.Join(parts[:i+1]...)
			stat, err := root.Lstat(current)
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			if err != nil {
				root.Close()
				return nil, nil, err
			}
			if stat.Mode()&os.ModeSymlink != 0 || (i == len(parts)-1 && !stat.Mode().IsRegular()) || (i < len(parts)-1 && !stat.IsDir()) {
				root.Close()
				return nil, nil, fmt.Errorf("refusing to delete unexpected file type at %q", current)
			}
		}
	}
	return root, paths, nil
}

func removeDataFiles(root *os.Root, paths []string) error {
	if root == nil {
		return nil
	}
	directories := map[string]bool{}
	for _, path := range paths {
		if err := root.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete %q: %w", path, err)
		}
		for dir := filepath.Dir(path); dir != "."; dir = filepath.Dir(dir) {
			directories[dir] = true
		}
	}
	ordered := make([]string, 0, len(directories))
	for dir := range directories {
		ordered = append(ordered, dir)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	// Remove only empty torrent directories; never recursively remove a save path.
	for _, dir := range ordered {
		_ = root.Remove(dir)
	}
	return nil
}

func (e *Engine) checkSharedFiles(entry torrentEntry, paths []string) error {
	directory, err := filepath.EvalSymlinks(entry.savePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return err
	}
	candidates := map[string]bool{}
	for _, path := range paths {
		candidates[filepath.Join(directory, path)] = true
	}
	for _, other := range e.torrents {
		if other.torrent == entry.torrent || other.torrent.Info() == nil {
			continue
		}
		otherDir, err := filepath.EvalSymlinks(other.savePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		otherDir, err = filepath.Abs(otherDir)
		if err != nil {
			return err
		}
		otherPaths, err := torrentFilePaths(other.torrent.Info())
		if err != nil {
			return err
		}
		for _, path := range otherPaths {
			if candidates[filepath.Join(otherDir, path)] {
				return fmt.Errorf("file %q is also used by another torrent", path)
			}
		}
	}
	return nil
}
