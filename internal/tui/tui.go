package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aayush/torrcli/internal/client"
	domain "github.com/aayush/torrcli/internal/model"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("15"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type model struct {
	ctx        context.Context
	daemon     *client.Client
	width      int
	height     int
	torrents   []domain.TorrentSnapshot
	selectedID domain.TorrentID
	loadError  error
}

type torrentsLoadedMsg struct {
	torrents []domain.TorrentSnapshot
	err      error
}

type refreshMsg time.Time

func Run(ctx context.Context, daemon *client.Client) error {
	_, err := tea.NewProgram(model{ctx: ctx, daemon: daemon}, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadTorrents(), refresh())
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.selectPrevious()
		case "down", "j":
			m.selectNext()
		}
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	case torrentsLoadedMsg:
		if message.err != nil {
			m.loadError = message.err
			return m, nil
		}
		m.torrents = message.torrents
		m.loadError = nil
		m.ensureSelection()
	case refreshMsg:
		return m, tea.Batch(m.loadTorrents(), refresh())
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	rows := []string{
		m.header(),
		mutedStyle.Render(strings.Repeat("─", m.width)),
		headerStyle.Render(m.tableHeader()),
	}
	rows = append(rows, m.tableRows()...)
	for len(rows) < max(0, m.height-3) {
		rows = append(rows, "")
	}
	rows = append(rows, mutedStyle.Render(strings.Repeat("─", m.width)), m.status(), mutedStyle.Render("↑/k up  ↓/j down  q quit"))
	return strings.Join(rows, "\n")
}

func (m model) loadTorrents() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		result, err := m.daemon.List(ctx)
		if err != nil {
			return torrentsLoadedMsg{err: err}
		}
		return torrentsLoadedMsg{torrents: result.Torrents}
	}
}

func refresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func (m *model) selectPrevious() {
	index := m.selectedIndex()
	if index > 0 {
		m.selectedID = m.torrents[index-1].ID
	}
}

func (m *model) selectNext() {
	index := m.selectedIndex()
	if index >= 0 && index < len(m.torrents)-1 {
		m.selectedID = m.torrents[index+1].ID
	}
}

func (m *model) ensureSelection() {
	if m.selectedIndex() >= 0 || len(m.torrents) == 0 {
		return
	}
	m.selectedID = m.torrents[0].ID
}

func (m model) selectedIndex() int {
	for index, torrent := range m.torrents {
		if torrent.ID == m.selectedID {
			return index
		}
	}
	return -1
}

func (m model) header() string {
	title := titleStyle.Render("torrcli")
	status := mutedStyle.Render("Connected to torrd")
	space := max(1, m.width-lipgloss.Width(title)-lipgloss.Width(status))
	return title + strings.Repeat(" ", space) + status
}

func (m model) status() string {
	if m.loadError != nil {
		return errorStyle.Render("Could not refresh torrents: " + m.loadError.Error())
	}
	return mutedStyle.Render(fmt.Sprintf("%d torrents", len(m.torrents)))
}

func (m model) tableHeader() string {
	nameWidth := max(16, min(42, m.width-62))
	return fmt.Sprintf("  %-*s %-12s %8s %10s %10s %14s", nameWidth, "NAME", "STATUS", "PROGRESS", "DOWN", "UP", "PEERS")
}

func (m model) tableRows() []string {
	if len(m.torrents) == 0 {
		return []string{mutedStyle.Render("  No torrents yet")}
	}

	nameWidth := max(16, min(42, m.width-62))
	visible := max(1, m.height-6)
	start := m.visibleStart(visible)
	end := min(len(m.torrents), start+visible)
	rows := make([]string, 0, end-start)
	for _, torrent := range m.torrents[start:end] {
		row := fmt.Sprintf("  %-*s %-12s %7.1f%% %10s %10s %2d (%ds/%dl)",
			nameWidth,
			truncate(torrent.Name, nameWidth),
			torrent.State,
			torrent.Progress*100,
			formatRate(torrent.DownloadRate),
			formatRate(torrent.UploadRate),
			torrent.ConnectedPeers,
			torrent.Seeders,
			torrent.Leechers,
		)
		if torrent.ID == m.selectedID {
			row = ">" + row[1:]
			rows = append(rows, selectedStyle.Width(m.width).Render(row))
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func (m model) visibleStart(visible int) int {
	selected := m.selectedIndex()
	if selected < visible {
		return 0
	}
	return min(selected-visible+1, len(m.torrents)-visible)
}

func formatRate(bytesPerSecond int64) string {
	units := []string{"B/s", "KiB/s", "MiB/s", "GiB/s"}
	rate := float64(bytesPerSecond)
	unit := 0
	for rate >= 1024 && unit < len(units)-1 {
		rate /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", rate, units[unit])
	}
	return fmt.Sprintf("%.1f %s", rate, units[unit])
}

func truncate(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
