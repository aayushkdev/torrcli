package tui

import (
	"context"
	"time"

	domain "github.com/aayush/torrcli/internal/model"
)

type dialogMode uint8

type focus uint8

const (
	focusTorrents focus = iota
	focusDetails
)

const (
	dialogNone dialogMode = iota
	dialogAdd
	dialogRemove
)

type model struct {
	ctx            context.Context
	daemon         daemonClient
	width          int
	height         int
	torrents       []domain.TorrentSnapshot
	selectedID     domain.TorrentID
	loadError      error
	loading        bool
	pending        bool
	notice         string
	noticeErr      bool
	noticeID       int
	dialog         dialogMode
	input          string
	details        domain.TorrentDetails
	showDetails    bool
	detailsLoading bool
	detailsTab     int
	focus          focus
	detailsErr     error
	selectedFile   int
}

type torrentsLoadedMsg struct {
	torrents []domain.TorrentSnapshot
	err      error
}

type refreshMsg time.Time

type actionResultMsg struct {
	action    string
	torrent   domain.TorrentSnapshot
	removedID domain.TorrentID
	err       error
}

type noticeClearedMsg int

type detailsLoadedMsg struct {
	details domain.TorrentDetails
	err     error
}

type filePriorityResultMsg struct {
	torrent  domain.TorrentSnapshot
	file     int
	priority domain.FilePriority
	err      error
}

type torrentsMovedMsg struct {
	torrents []domain.TorrentSnapshot
	err      error
}
