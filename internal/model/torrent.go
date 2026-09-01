package model

import "time"

type TorrentID string

type TorrentState string

const (
	TorrentStateFetchingMetadata TorrentState = "fetching_metadata"
	TorrentStateChecking         TorrentState = "checking"
	TorrentStateQueued           TorrentState = "queued"
	TorrentStateDownloading      TorrentState = "downloading"
	TorrentStateSeeding          TorrentState = "seeding"
	TorrentStatePaused           TorrentState = "paused"
	TorrentStateError            TorrentState = "error"
)

type FilePriority string

const (
	FilePrioritySkip   FilePriority = "skip"
	FilePriorityNormal FilePriority = "normal"
	FilePriorityHigh   FilePriority = "high"
)

type AddInput struct {
	Source         string
	SavePath       string
	FilePriorities map[int]FilePriority
}

type TorrentSnapshot struct {
	ID             TorrentID
	Name           string
	State          TorrentState
	Progress       float64
	DownloadRate   int64
	UploadRate     int64
	ConnectedPeers int
	Seeders        int
	Leechers       int
	AddedAt        time.Time
	Error          string
}

type TorrentDetails struct {
	Torrent TorrentSnapshot `json:"torrent"`
	Files   []FileSnapshot  `json:"files"`
}

type EngineEventType string

const (
	EngineEventUpdated EngineEventType = "updated"
	EngineEventError   EngineEventType = "error"
	EngineEventRemoved EngineEventType = "removed"
)

type EngineEvent struct {
	Type     EngineEventType
	Torrent  TorrentSnapshot
	Occurred time.Time
}
