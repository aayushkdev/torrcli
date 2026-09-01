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
	ETASeconds     int64
	ConnectedPeers int
	Seeders        int
	Leechers       int
	AddedAt        time.Time
	Error          string
}

type TorrentDetails struct {
	Torrent TorrentSnapshot `json:"torrent"`
	Info    TorrentInfo     `json:"info"`
	Files   []FileSnapshot  `json:"files"`
}

type TorrentInfo struct {
	SavePath    string `json:"save_path"`
	TotalSize   int64  `json:"total_size"`
	PieceLength int64  `json:"piece_length"`
	PieceCount  int    `json:"piece_count"`
	InfoHashV1  string `json:"info_hash_v1"`
	InfoHashV2  string `json:"info_hash_v2"`
	CreatedBy   string `json:"created_by"`
	Comment     string `json:"comment"`
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
