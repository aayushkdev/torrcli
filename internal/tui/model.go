package tui

import (
	"context"
	"time"

	domain "github.com/aayush/torrcli/internal/model"
)

type dialogMode uint8

const (
	dialogNone dialogMode = iota
	dialogAdd
	dialogRemove
)

type model struct {
	ctx        context.Context
	daemon     daemonClient
	width      int
	height     int
	torrents   []domain.TorrentSnapshot
	selectedID domain.TorrentID
	loadError  error
	pending    bool
	notice     string
	noticeErr  bool
	noticeID   int
	dialog     dialogMode
	input      string
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
