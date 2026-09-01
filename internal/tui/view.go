package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	domain "github.com/aayush/torrcli/internal/model"
)

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	rows := []string{
		m.header(),
		m.paneDivider(" Torrents ", m.focus == focusTorrents),
		headerStyle.Render(m.tableHeader()),
	}
	rows = append(rows, m.tableRows()...)
	for len(rows) < m.mainTableHeight() {
		rows = append(rows, "")
	}
	rows = append(rows, m.paneDivider(" Details ", m.focus == focusDetails))
	rows = append(rows, strings.Split(m.detailsPanel(m.detailsPaneHeight()), "\n")...)
	rows = append(rows, mutedStyle.Render(strings.Repeat("─", m.width)), m.status(), mutedStyle.Render("a add  d remove  shift+↑/↓ move  ↑/k up  ↓/j down  space pause/resume  q quit"))
	screen := strings.Join(rows, "\n")
	if m.dialog == dialogNone {
		return screen
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogStyle.Render(m.dialogView()))
}

func (m model) detailsPanel(height int) string {
	rows := []string{}
	if m.detailsLoading {
		rows = append(rows, m.detailsTabs(), mutedStyle.Render("Loading details…"))
		return fillRows(rows, height)
	}
	if m.detailsErr != nil {
		rows = append(rows, m.detailsTabs(), errorStyle.Render("Could not load details: "+m.detailsErr.Error()))
		return fillRows(rows, height)
	}
	if m.details.Torrent.ID == "" {
		return fillRows([]string{m.detailsTabs(), mutedStyle.Render("Select a torrent to view details")}, height)
	}
	if m.detailsTab == 0 {
		t := m.details.Torrent
		rows = append(rows,
			m.detailsTabs(),
			fmt.Sprintf("State: %s", t.State),
			fmt.Sprintf("Progress: %.1f%%", t.Progress*100),
			fmt.Sprintf("Peers: %d (%d seeders, %d leechers)", t.ConnectedPeers, t.Seeders, t.Leechers),
			fmt.Sprintf("Download: %s   Upload: %s", formatRate(t.DownloadRate), formatRate(t.UploadRate)),
		)
		return fillRows(rows, height)
	}
	if len(m.details.Files) == 0 {
		return fillRows([]string{m.detailsTabs(), mutedStyle.Render("Metadata is not available yet")}, height)
	}
	rows = append(rows, m.detailsTabs(), headerStyle.Render("  FILE                                      PROGRESS  PRIORITY"))
	nameWidth := max(12, min(40, m.width-24))
	rows[len(rows)-1] = headerStyle.Render(fmt.Sprintf("  %-*s %s %s", nameWidth, "FILE", centerText("PROGRESS", 9), centerText("PRIORITY", 8)))
	for _, file := range m.details.Files[:min(len(m.details.Files), height-2)] {
		progress := float64(file.Completed) * 100 / float64(max(1, int(file.Length)))
		rows = append(rows, fmt.Sprintf("  %-*s %s %s", nameWidth, truncate(file.Path, nameWidth), centerText(fmt.Sprintf("%.1f%%", progress), 9), centerText(string(file.Priority), 8)))
	}
	return fillRows(rows, height)
}

func (m model) paneDivider(name string, active bool) string {
	label := mutedStyle.Render(name)
	if active {
		label = focusStyle.Render("▸" + name)
	}
	return label + mutedStyle.Render(strings.Repeat("─", max(0, m.width-lipgloss.Width(label))))
}

func (m model) detailsTabs() string {
	overview, files := "Overview", "Files"
	if m.detailsTab == 0 {
		overview = "[Overview]"
	} else {
		files = "[Files]"
	}
	return mutedStyle.Render(overview + "  " + files)
}

func fillRows(rows []string, height int) string {
	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

func (m model) detailsPaneHeight() int { return max(6, m.height/3) }

func (m model) mainTableHeight() int { return max(3, m.height-m.detailsPaneHeight()-4) }

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
	visible := max(1, m.mainTableHeight()-3)
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
	if m.width < 70 {
		return title
	}
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
	if m.width < 70 {
		nameWidth := max(12, m.width-23)
		return fmt.Sprintf("  %-*s %s %s", nameWidth, "NAME", centerText("PROGRESS", 9), centerText("STATUS", 10))
	}
	if m.width < 80 {
		return fmt.Sprintf("  %-*s %-12s %8s %10s %10s", max(16, m.width-46), "NAME", "STATUS", "PROGRESS", "DOWN", "UP")
	}
	nameWidth := max(16, min(42, m.width-62))
	return fmt.Sprintf("  %-*s %s %s %s %s %s", nameWidth, "NAME", centerText("STATUS", 12), centerText("PROGRESS", 8), centerText("DOWN", 10), centerText("UP", 10), centerText("PEERS", 14))
}

func (m model) tableRows() []string {
	if len(m.torrents) == 0 {
		if m.loading {
			return []string{mutedStyle.Render("  Loading torrents…")}
		}
		return []string{mutedStyle.Render("  No torrents yet")}
	}

	nameWidth := max(16, min(42, m.width-62))
	visible := max(1, m.height-9)
	start := m.visibleStart(visible)
	end := min(len(m.torrents), start+visible)
	rows := make([]string, 0, end-start)
	for _, torrent := range m.torrents[start:end] {
		row := m.torrentRow(torrent, nameWidth)
		if torrent.ID == m.selectedID {
			row = ">" + row[1:]
			rows = append(rows, selectedStyle.Width(m.width).Render(row))
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func (m model) torrentRow(torrent domain.TorrentSnapshot, nameWidth int) string {
	if m.width < 70 {
		return fmt.Sprintf("  %-*s %s %s", nameWidth, truncate(torrent.Name, nameWidth), centerText(fmt.Sprintf("%.1f%%", torrent.Progress*100), 9), centerText(string(torrent.State), 10))
	}
	if m.width < 80 {
		return fmt.Sprintf("  %-*s %-12s %7.1f%% %10s %10s", nameWidth, truncate(torrent.Name, nameWidth), torrent.State, torrent.Progress*100, formatRate(torrent.DownloadRate), formatRate(torrent.UploadRate))
	}
	peers := fmt.Sprintf("%d (%ds/%dl)", torrent.ConnectedPeers, torrent.Seeders, torrent.Leechers)
	return fmt.Sprintf("  %-*s %s %s %s %s %s", nameWidth, truncate(torrent.Name, nameWidth), centerText(string(torrent.State), 12), centerText(fmt.Sprintf("%.1f%%", torrent.Progress*100), 8), centerText(formatRate(torrent.DownloadRate), 10), centerText(formatRate(torrent.UploadRate), 10), centerText(peers, 14))
}

func centerText(value string, width int) string {
	if len(value) >= width {
		return value
	}
	left := (width - len(value)) / 2
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", width-left-len(value))
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
