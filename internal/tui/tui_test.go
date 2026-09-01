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

func TestModelRendersDetails(t *testing.T) {
	m := model{width: 100, height: 24, detailsTab: 1, details: domain.TorrentDetails{Torrent: domain.TorrentSnapshot{ID: "one", Name: "example", Progress: 0.5}, Files: []domain.FileSnapshot{{Path: "folder/file.iso", Length: 2048, Completed: 1024, Priority: domain.FilePriorityHigh}}}}
	for _, value := range []string{"folder/file.iso", "50.0%", "high", "Files"} {
		if !strings.Contains(m.View(), value) {
			t.Errorf("details view does not contain %q", value)
		}
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
