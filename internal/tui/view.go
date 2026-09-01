package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.showDetails {
		return m.detailsView()
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
	rows = append(rows, mutedStyle.Render(strings.Repeat("─", m.width)), m.status(), mutedStyle.Render("a add  d remove  shift+↑/↓ move  ↑/k up  ↓/j down  space pause/resume  q quit"))
	screen := strings.Join(rows, "\n")
	if m.dialog == dialogNone {
		return screen
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogStyle.Render(m.dialogView()))
}

func (m model) detailsView() string {
	if m.detailsErr != nil {
		return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("Torrent details"), errorStyle.Render("Could not load details: "+m.detailsErr.Error()), mutedStyle.Render("esc back"))
	}
	torrent := m.details.Torrent
	rows := []string{
		titleStyle.Render(torrent.Name),
		mutedStyle.Render("ID " + string(torrent.ID)),
		mutedStyle.Render(strings.Repeat("─", m.width)),
		fmt.Sprintf("%s  %.1f%%  ↓%s  ↑%s  %d peers", torrent.State, torrent.Progress*100, formatRate(torrent.DownloadRate), formatRate(torrent.UploadRate), torrent.ConnectedPeers),
		"",
		headerStyle.Render("  FILE                                      PROGRESS       SIZE       PRIORITY"),
	}
	if len(m.details.Files) == 0 {
		rows = append(rows, mutedStyle.Render("  Metadata is not available yet"))
	}
	visible := max(1, m.height-9)
	start := m.detailsStart(visible)
	end := min(len(m.details.Files), start+visible)
	for index, file := range m.details.Files[start:end] {
		progress := 0.0
		if file.Length > 0 {
			progress = float64(file.Completed) / float64(file.Length) * 100
		}
		row := fmt.Sprintf("  %-40s %7.1f%% %10s  %s", truncate(file.Path, 40), progress, formatBytes(file.Length), file.Priority)
		if start+index == m.selectedFile {
			row = ">" + row[1:]
			rows = append(rows, selectedStyle.Width(m.width).Render(row))
			continue
		}
		rows = append(rows, row)
	}
	for len(rows) < max(0, m.height-2) {
		rows = append(rows, "")
	}
	rows = append(rows, mutedStyle.Render(strings.Repeat("─", m.width)), mutedStyle.Render("↑/k up  ↓/j down  space change priority  esc back"))
	return strings.Join(rows, "\n")
}

func (m model) detailsStart(visible int) int {
	if m.selectedFile < visible {
		return 0
	}
	return min(m.selectedFile-visible+1, len(m.details.Files)-visible)
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
	if m.notice != "" {
		if m.noticeErr {
			return errorStyle.Render(m.notice)
		}
		return mutedStyle.Render(m.notice)
	}
	return mutedStyle.Render(fmt.Sprintf("%d torrents", len(m.torrents)))
}

func (m model) dialogView() string {
	switch m.dialog {
	case dialogAdd:
		inputWidth := max(20, min(72, m.width-12))
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Add torrent"),
			mutedStyle.Render("Paste a magnet link or .torrent path"),
			"> "+truncate(m.input, inputWidth)+"█",
			mutedStyle.Render("enter add  esc cancel"),
		)
	case dialogRemove:
		name := "selected torrent"
		if index := m.selectedIndex(); index >= 0 {
			name = m.torrents[index].Name
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Remove torrent?"),
			truncate(name, max(20, min(72, m.width-12))),
			mutedStyle.Render("Downloaded files will be kept."),
			mutedStyle.Render("enter remove  esc cancel"),
		)
	default:
		return ""
	}
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
		row := fmt.Sprintf("  %-*s %-12s %7.1f%% %10s %10s %2d (%ds/%dl)", nameWidth, truncate(torrent.Name, nameWidth), torrent.State, torrent.Progress*100, formatRate(torrent.DownloadRate), formatRate(torrent.UploadRate), torrent.ConnectedPeers, torrent.Seeders, torrent.Leechers)
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

func formatBytes(bytes int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
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
