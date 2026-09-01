package anacrolix

import (
	"context"
	"fmt"
	"math"
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
	torrent          *torrent.Torrent
	storage          storage.ClientImplCloser
	addedAt          time.Time
	paused           bool
	lastDownload     int64
	lastUpload       int64
	lastDownloadRate int64
	lastUploadRate   int64
	lastSample       time.Time
}

func New() (*Engine, error) {
	config := torrent.NewDefaultClientConfig()
	config.DefaultStorage = storage.NewFileOpts(storage.NewFileClientOpts{})
	client, err := torrent.NewClient(config)
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
	go func() {
		select {
		case <-added.GotInfo():
			added.DownloadAll()
			for index, priority := range input.FilePriorities {
				files := added.Files()
				if index >= 0 && index < len(files) {
					files[index].SetPriority(piecePriority(priority))
				}
			}
		case <-e.client.Closed():
		}
	}()
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
	entry.torrent.DisallowDataUpload()
	for _, peer := range entry.torrent.PeerConns() {
		_ = peer.Close()
	}
	for _, peer := range entry.torrent.WebseedPeerConns() {
		_ = peer.Close()
	}
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
	entry.torrent.AllowDataUpload()
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
	if entry.torrent.Info() == nil {
		return fmt.Errorf("torrent metadata is not available yet")
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
	return e.snapshotLocked(id, entry), nil
}

func (e *Engine) Details(ctx context.Context, id model.TorrentID) (model.TorrentDetails, error) {
	if err := ctx.Err(); err != nil {
		return model.TorrentDetails{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entry(id)
	if err != nil {
		return model.TorrentDetails{}, err
	}
	details := model.TorrentDetails{Torrent: snapshot(id, entry, entry.lastDownloadRate, entry.lastUploadRate)}
	if entry.torrent.Info() == nil {
		return details, nil
	}
	for index, file := range entry.torrent.Files() {
		details.Files = append(details.Files, model.FileSnapshot{Index: index, Path: file.DisplayPath(), Length: file.Length(), Completed: file.BytesCompleted(), Priority: filePriority(file.Priority())})
	}
	return details, nil
}

func (e *Engine) snapshotLocked(id model.TorrentID, entry torrentEntry) model.TorrentSnapshot {
	now := time.Now()
	stats := entry.torrent.Stats()
	downloaded := stats.BytesReadUsefulData.Int64()
	uploaded := stats.BytesWrittenData.Int64()
	downloadRate, uploadRate := int64(0), int64(0)
	if !entry.lastSample.IsZero() {
		seconds := now.Sub(entry.lastSample).Seconds()
		if seconds > 0 {
			downloadRate = int64(float64(downloaded-entry.lastDownload) / seconds)
			uploadRate = int64(float64(uploaded-entry.lastUpload) / seconds)
		}
	}
	entry.lastDownload, entry.lastUpload, entry.lastSample = downloaded, uploaded, now
	entry.lastDownloadRate, entry.lastUploadRate = downloadRate, uploadRate
	e.torrents[id] = entry
	return snapshot(id, entry, downloadRate, uploadRate)
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

func filePriority(priority torrent.PiecePriority) model.FilePriority {
	switch priority {
	case torrent.PiecePriorityNone:
		return model.FilePrioritySkip
	case torrent.PiecePriorityHigh:
		return model.FilePriorityHigh
	default:
		return model.FilePriorityNormal
	}
}

func snapshot(id model.TorrentID, entry torrentEntry, downloadRate, uploadRate int64) model.TorrentSnapshot {
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
	etaSeconds := int64(0)
	if remaining := length - completed; remaining > 0 && downloadRate > 0 {
		etaSeconds = int64(math.Ceil(float64(remaining) / float64(downloadRate)))
	}
	return model.TorrentSnapshot{
		ID:             id,
		Name:           torrent.Name(),
		State:          state,
		Progress:       progress,
		DownloadRate:   downloadRate,
		UploadRate:     uploadRate,
		ETASeconds:     etaSeconds,
		ConnectedPeers: stats.ActivePeers,
		Seeders:        stats.ConnectedSeeders,
		Leechers:       stats.ActivePeers - stats.ConnectedSeeders,
		AddedAt:        entry.addedAt,
	}
}
