package anacrolix

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"

	"github.com/aayush/torrcli/internal/model"
)

type Engine struct {
	client *torrent.Client

	mu       sync.Mutex
	torrents map[model.TorrentID]torrentEntry
	events   chan model.EngineEvent
}

type torrentEntry struct {
	torrent *torrent.Torrent
	storage storage.ClientImplCloser
	addedAt time.Time
	paused  bool
}

func New() (*Engine, error) {
	client, err := torrent.NewClient(torrent.NewDefaultClientConfig())
	if err != nil {
		return nil, fmt.Errorf("create torrent client: %w", err)
	}
	return &Engine{
		client:   client,
		torrents: make(map[model.TorrentID]torrentEntry),
		events:   make(chan model.EngineEvent),
	}, nil
}

func (e *Engine) Add(ctx context.Context, input model.AddInput) (model.TorrentID, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if input.SavePath == "" {
		return "", false, fmt.Errorf("torrent save path is required")
	}
	if err := os.MkdirAll(input.SavePath, 0o755); err != nil {
		return "", false, fmt.Errorf("create torrent save path: %w", err)
	}

	spec, err := torrentSpec(input.Source)
	if err != nil {
		return "", false, err
	}
	store := storage.NewFileOpts(storage.NewFileClientOpts{ClientBaseDir: input.SavePath})
	spec.Storage = store

	e.mu.Lock()
	defer e.mu.Unlock()
	added, created, err := e.client.AddTorrentSpec(spec)
	if err != nil {
		_ = store.Close()
		return "", false, fmt.Errorf("add torrent: %w", err)
	}

	id := model.TorrentID(added.InfoHash().HexString())
	if !created {
		_ = store.Close()
		return id, false, nil
	}
	e.torrents[id] = torrentEntry{torrent: added, storage: store, addedAt: time.Now()}
	return id, true, nil
}

func (e *Engine) Pause(ctx context.Context, id model.TorrentID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return err
	}
	entry.torrent.DisallowDataDownload()
	entry.paused = true
	e.torrents[id] = entry
	return nil
}

func (e *Engine) Resume(ctx context.Context, id model.TorrentID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return err
	}
	entry.torrent.AllowDataDownload()
	entry.paused = false
	e.torrents[id] = entry
	return nil
}

func (e *Engine) Remove(ctx context.Context, id model.TorrentID, deleteData bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deleteData {
		return fmt.Errorf("deleting torrent data is not implemented")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return err
	}
	entry.torrent.Drop()
	if err := entry.storage.Close(); err != nil {
		return fmt.Errorf("close torrent storage: %w", err)
	}
	delete(e.torrents, id)
	return nil
}

func (e *Engine) SetFilePriority(ctx context.Context, id model.TorrentID, fileIndex int, priority model.FilePriority) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return err
	}
	files := entry.torrent.Files()
	if fileIndex < 0 || fileIndex >= len(files) {
		return fmt.Errorf("file %d not found in torrent %q", fileIndex, id)
	}
	files[fileIndex].SetPriority(piecePriority(priority))
	return nil
}

func (e *Engine) Snapshot(ctx context.Context, id model.TorrentID) (model.TorrentSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.TorrentSnapshot{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return model.TorrentSnapshot{}, err
	}
	return snapshot(id, entry), nil
}

func (e *Engine) Events() <-chan model.EngineEvent {
	return e.events
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, entry := range e.torrents {
		entry.torrent.Drop()
		if err := entry.storage.Close(); err != nil {
			return fmt.Errorf("close torrent storage: %w", err)
		}
		delete(e.torrents, id)
	}
	close(e.events)
	errs := e.client.Close()
	if len(errs) > 0 {
		return fmt.Errorf("close torrent client: %w", errs[0])
	}
	return nil
}

func (e *Engine) entry(id model.TorrentID) (torrentEntry, error) {
	entry, ok := e.torrents[id]
	if !ok {
		return torrentEntry{}, fmt.Errorf("torrent %q not found", id)
	}
	return entry, nil
}

func torrentSpec(source string) (*torrent.TorrentSpec, error) {
	if strings.HasPrefix(strings.ToLower(source), "magnet:") {
		spec, err := torrent.TorrentSpecFromMagnetUri(source)
		if err != nil {
			return nil, fmt.Errorf("parse magnet link: %w", err)
		}
		return spec, nil
	}
	meta, err := metainfo.LoadFromFile(source)
	if err != nil {
		return nil, fmt.Errorf("load torrent file: %w", err)
	}
	spec, err := torrent.TorrentSpecFromMetaInfoErr(meta)
	if err != nil {
		return nil, fmt.Errorf("read torrent file: %w", err)
	}
	return spec, nil
}

func piecePriority(priority model.FilePriority) torrent.PiecePriority {
	switch priority {
	case model.FilePrioritySkip:
		return torrent.PiecePriorityNone
	case model.FilePriorityHigh:
		return torrent.PiecePriorityHigh
	default:
		return torrent.PiecePriorityNormal
	}
}

func snapshot(id model.TorrentID, entry torrentEntry) model.TorrentSnapshot {
	torrent := entry.torrent
	completed := torrent.BytesCompleted()
	length := torrent.Length()
	progress := 0.0
	if length > 0 {
		progress = float64(completed) / float64(length)
	}
	state := model.TorrentStateDownloading
	if entry.paused {
		state = model.TorrentStatePaused
	} else if torrent.Info() == nil {
		state = model.TorrentStateFetchingMetadata
	} else if completed >= length {
		state = model.TorrentStateSeeding
	}
	stats := torrent.Stats()
	return model.TorrentSnapshot{
		ID:             id,
		Name:           torrent.Name(),
		State:          state,
		Progress:       progress,
		ConnectedPeers: stats.ActivePeers,
		AddedAt:        entry.addedAt,
	}
}
