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
	for _, value := range []string{"example.iso", "downloading", "50.0%", "1.5 KiB/s", "2 (1s/1l)", "Connected to torrd"} {
		if !strings.Contains(view, value) {
			t.Errorf("view does not contain %q: %s", value, view)
		}
	}
}
