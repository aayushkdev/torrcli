package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	domain "github.com/aayush/torrcli/internal/model"
)

func TestModelHandlesWindowSize(t *testing.T) {
	updated, _ := model{}.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	actual := updated.(model)
	if actual.width != 80 || actual.height != 24 {
		t.Fatalf("model size = %dx%d", actual.width, actual.height)
	}
}

func TestModelSelectsAndRetainsTorrent(t *testing.T) {
	m := model{torrents: []domain.TorrentSnapshot{{ID: "first"}, {ID: "second"}}}
	m.ensureSelection()
	if m.selectedID != "first" {
		t.Fatalf("selected ID = %q", m.selectedID)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.selectedID != "second" {
		t.Fatalf("selected ID = %q", m.selectedID)
	}

	updated, _ = m.Update(torrentsLoadedMsg{torrents: []domain.TorrentSnapshot{{ID: "second"}}})
	m = updated.(model)
	if m.selectedID != "second" {
		t.Fatalf("selected ID after refresh = %q", m.selectedID)
	}
}

func TestModelListsTorrents(t *testing.T) {
	m := model{ctx: context.Background(), width: 100, torrents: []domain.TorrentSnapshot{{ID: "one", Name: "example.iso", State: domain.TorrentStateDownloading, Progress: 0.5, DownloadRate: 1536, ConnectedPeers: 2, Seeders: 1, Leechers: 1}}}
	m.ensureSelection()
	view := m.View()
	for _, value := range []string{"example.iso", "downloading", "50.0%", "1.5 KiB/s", "2 (1s/1l)", "Connected to torrd", "space pause/resume"} {
		if !strings.Contains(view, value) {
			t.Errorf("view does not contain %q: %s", value, view)
		}
	}
}

func TestModelUpdatesTorrentAfterAction(t *testing.T) {
	m := model{torrents: []domain.TorrentSnapshot{{ID: "one", State: domain.TorrentStateDownloading}}}
	updated, _ := m.Update(actionResultMsg{torrent: domain.TorrentSnapshot{ID: "one", State: domain.TorrentStatePaused}, action: "pause"})
	m = updated.(model)
	if m.torrents[0].State != domain.TorrentStatePaused {
		t.Fatalf("torrent state = %q", m.torrents[0].State)
	}
	if m.notice != "Pause" {
		t.Fatalf("notice = %q", m.notice)
	}
}

func TestModelOpensAndCancelsDialogs(t *testing.T) {
	m := model{torrents: []domain.TorrentSnapshot{{ID: "one", Name: "example"}}, selectedID: "one"}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(model)
	if m.dialog != dialogAdd {
		t.Fatalf("dialog = %d", m.dialog)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.dialog != dialogNone {
		t.Fatalf("dialog = %d", m.dialog)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(model)
	if m.dialog != dialogRemove {
		t.Fatalf("dialog = %d", m.dialog)
	}
}

func TestActionsDialogKeepsTorrentListVisible(t *testing.T) {
	m := model{width: 100, height: 24, torrents: []domain.TorrentSnapshot{{ID: "one", Name: "example.iso"}}, selectedID: "one", dialog: dialogActions}
	view := m.View()
	for _, value := range []string{"example.iso", "Torrent actions", "Force recheck"} {
		if !strings.Contains(view, value) {
			t.Errorf("dialog view does not contain %q: %s", value, view)
		}
	}
}

func TestFindPeersIsDisabledWithoutMetadata(t *testing.T) {
	m := model{torrents: []domain.TorrentSnapshot{{ID: "one", State: domain.TorrentStateFetchingMetadata}}, selectedID: "one", dialog: dialogActions, actionIndex: 2}
	if m.actionEnabled(actionFindPeers) {
		t.Fatal("find peers is enabled while metadata is unavailable")
	}
	updated, command := m.updateDialog(tea.KeyMsg{Type: tea.KeyEnter})
	actual := updated.(model)
	if command != nil || actual.dialog != dialogActions {
		t.Fatal("disabled action was executed")
	}
}

func TestModelRendersDetails(t *testing.T) {
	m := model{width: 100, height: 24, detailsTab: 1, details: domain.TorrentDetails{Torrent: domain.TorrentSnapshot{ID: "one", Name: "example", Progress: 0.5}, Files: []domain.FileSnapshot{{Path: "folder/file.iso", Length: 2048, Completed: 1024, Priority: domain.FilePriorityHigh}}}}
	for _, value := range []string{"folder/file.iso", "50.0%", "2.0 KiB", "high", "Content"} {
		if !strings.Contains(m.View(), value) {
			t.Errorf("details view does not contain %q", value)
		}
	}
}

func TestModelRendersOverview(t *testing.T) {
	m := model{
		width:  120,
		height: 30,
		details: domain.TorrentDetails{
			Torrent: domain.TorrentSnapshot{ID: "one", State: domain.TorrentStateDownloading, Progress: 0.5, DownloadRate: 2048, UploadRate: 1024, ETASeconds: 60, ConnectedPeers: 2, Seeders: 1, Leechers: 1},
			Info:    domain.TorrentInfo{SavePath: "/downloads", TotalSize: 4096, PieceLength: 1024, PieceCount: 4, InfoHashV1: "v1hash", InfoHashV2: "v2hash", CreatedBy: "torrent maker", Comment: "example comment"},
			Files:   []domain.FileSnapshot{{Path: "example.iso", Length: 4096, Completed: 2048}},
		},
	}
	view := m.View()
	for _, value := range []string{"LIVE ACTIVITY", "Downloaded: 2.0 KiB / 4.0 KiB", "Torrent information", "Save path: /downloads", "Hash v1: v1hash", "Created by: torrent maker", "Comment: example comment"} {
		if !strings.Contains(view, value) {
			t.Errorf("overview does not contain %q: %s", value, view)
		}
	}
}

func TestModelRefreshesSelectedOverviewSnapshot(t *testing.T) {
	m := model{
		selectedID: "one",
		details:    domain.TorrentDetails{Torrent: domain.TorrentSnapshot{ID: "one", DownloadRate: 1024}},
	}
	updated, _ := m.Update(torrentsLoadedMsg{torrents: []domain.TorrentSnapshot{{ID: "one", DownloadRate: 2048}}})
	m = updated.(model)
	if m.details.Torrent.DownloadRate != 2048 {
		t.Fatalf("details download rate = %d", m.details.Torrent.DownloadRate)
	}
}

func TestNextPriority(t *testing.T) {
	if actual := nextPriority(domain.FilePrioritySkip); actual != domain.FilePriorityNormal {
		t.Fatalf("skip next = %q", actual)
	}
	if actual := nextPriority(domain.FilePriorityNormal); actual != domain.FilePriorityHigh {
		t.Fatalf("normal next = %q", actual)
	}
	if actual := nextPriority(domain.FilePriorityHigh); actual != domain.FilePrioritySkip {
		t.Fatalf("high next = %q", actual)
	}
}
