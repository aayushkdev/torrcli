package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/aayush/torrcli/internal/rpc"
	"github.com/anacrolix/torrent/metainfo"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	addSource = iota
	addPath
	addSequential
	addPaused
	addSubmit
)

type addForm struct {
	field       int
	cancel      bool
	cursors     [2]int
	path        string
	defaultPath string
	sequential  bool
	paused      bool
	preview     string
	err         string
}

type addDefaultsMsg struct{ path string }
type addPreviewMsg struct{ source, text string }

func (m model) loadAddDefaults() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		info, err := m.daemon.Info(ctx)
		if err != nil {
			return addDefaultsMsg{}
		}
		return addDefaultsMsg{path: info.DownloadDirectory}
	}
}

func (m model) previewAddSource() tea.Cmd {
	source := strings.TrimSpace(m.input)
	return func() tea.Msg {
		result := addPreviewMsg{source: source}
		if source == "" {
			return result
		}
		if strings.HasPrefix(strings.ToLower(source), "magnet:") {
			magnet, err := metainfo.ParseMagnetV2Uri(source)
			if err != nil {
				result.text = "Invalid magnet link"
				return result
			}
			result.text = "Size and files available after metadata loads"
			if magnet.DisplayName != "" {
				result.text = magnet.DisplayName + " · " + result.text
			}
			return result
		}
		path, err := localPath(source)
		if err != nil {
			result.text = err.Error()
			return result
		}
		meta, err := metainfo.LoadFromFile(path)
		if err != nil {
			result.text = "Could not read torrent file"
			return result
		}
		info, err := meta.UnmarshalInfo()
		if err != nil {
			result.text = "Invalid torrent file"
			return result
		}
		var size int64
		files := info.UpvertedFiles()
		for _, file := range files {
			size += file.Length
		}
		fileLabel := "files"
		if len(files) == 1 {
			fileLabel = "file"
		}
		result.text = fmt.Sprintf("%s · %s · %d %s", info.BestName(), formatBytes(size), len(files), fileLabel)
		return result
	}
}

func localPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Abs(path)
}

func (m model) updateAddDialog(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlC {
		if !m.pending && m.addForm.field == addSource && m.input != "" {
			m.input = ""
			m.addForm.cursors[addSource] = 0
			m.addForm.preview = ""
			m.addForm.err = ""
			return m, nil
		}
		if !m.pending && m.addForm.field == addPath && m.addForm.path != "" {
			m.addForm.path = ""
			m.addForm.cursors[addPath] = 0
			m.addForm.err = ""
			return m, nil
		}
		m.dialog = dialogNone
		m.input = ""
		m.addForm = addForm{}
		return m, nil
	}

	if m.pending {
		return m, nil
	}
	switch key.Type {
	case tea.KeyEsc:
		m.dialog = dialogNone
		m.input = ""
		return m, nil
	case tea.KeyUp, tea.KeyShiftTab, tea.KeyDown, tea.KeyTab:
		old := m.addForm.field
		delta := 1
		if key.Type == tea.KeyUp || key.Type == tea.KeyShiftTab {
			delta = -1
		}
		m.addForm.field = (old + delta + addSubmit + 1) % (addSubmit + 1)
		m.addForm.cancel = false
		if old == addSource {
			return m, m.previewAddSource()
		}
		return m, nil
	case tea.KeyEnter:
		switch m.addForm.field {
		case addSource, addPath:
			m.addForm.field++
			return m, m.previewAddSource()
		case addSequential:
			m.addForm.sequential = !m.addForm.sequential
		case addPaused:
			m.addForm.paused = !m.addForm.paused
		case addSubmit:
			if m.addForm.cancel {
				m.dialog = dialogNone
				m.input = ""
				m.addForm = addForm{}
				return m, nil
			}
			source := strings.TrimSpace(m.input)
			if source == "" {
				m.addForm.err = "Enter a magnet link or .torrent path"
				m.addForm.field = addSource
				return m, nil
			}
			if !strings.HasPrefix(strings.ToLower(source), "magnet:") {
				path, err := localPath(source)
				if err != nil {
					m.addForm.err = err.Error()
					return m, nil
				}
				source = path
			}
			path := strings.TrimSpace(m.addForm.path)
			if path != "" {
				absolute, err := localPath(path)
				if err != nil {
					m.addForm.err = err.Error()
					return m, nil
				}
				path = absolute
			}
			m.addForm.err = ""
			cmd := m.addTorrent(rpc.AddTorrentParams{Source: source, SavePath: path, Sequential: m.addForm.sequential, Paused: m.addForm.paused})
			return m, cmd
		}
		return m, nil
	}
	if m.addForm.field == addSubmit && (key.Type == tea.KeyLeft || key.Type == tea.KeyRight) {
		m.addForm.cancel = !m.addForm.cancel
		return m, nil
	}
	if m.addForm.field == addSequential || m.addForm.field == addPaused {
		if key.Type == tea.KeySpace || key.Type == tea.KeyLeft || key.Type == tea.KeyRight {
			if m.addForm.field == addSequential {
				m.addForm.sequential = !m.addForm.sequential
			} else {
				m.addForm.paused = !m.addForm.paused
			}
		}
		return m, nil
	}
	if m.addForm.field > addPath {
		return m, nil
	}
	value := &m.input
	if m.addForm.field == addPath {
		value = &m.addForm.path
	}
	runes := []rune(*value)
	cursor := min(m.addForm.cursors[m.addForm.field], len(runes))
	switch key.Type {
	case tea.KeyLeft:
		cursor = max(0, cursor-1)
	case tea.KeyRight:
		cursor = min(len(runes), cursor+1)
	case tea.KeyHome, tea.KeyCtrlA:
		cursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		cursor = len(runes)
	case tea.KeyBackspace:
		if cursor > 0 {
			runes = append(runes[:cursor-1], runes[cursor:]...)
			cursor--
		}
	case tea.KeyDelete:
		if cursor < len(runes) {
			runes = append(runes[:cursor], runes[cursor+1:]...)
		}
	case tea.KeyCtrlU:
		runes = runes[cursor:]
		cursor = 0
	case tea.KeyRunes, tea.KeySpace:
		incoming := key.Runes
		if key.Type == tea.KeySpace {
			incoming = []rune{' '}
		}
		clean := []rune{}
		for _, r := range incoming {
			if !unicode.IsControl(r) {
				clean = append(clean, r)
			}
		}
		tail := append([]rune(nil), runes[cursor:]...)
		runes = append(append(runes[:cursor], clean...), tail...)
		cursor += len(clean)
	}
	*value = string(runes)
	m.addForm.cursors[m.addForm.field] = cursor
	m.addForm.err = ""
	if m.addForm.field == addSource {
		m.addForm.preview = ""
	}
	return m, nil
}

// Keep the insertion point visible without removing text that already fits.
func addInputWindow(value string, cursor, width int) string {
	runes := []rune(value)
	cursor = max(0, min(cursor, len(runes)))
	before := string(runes[:cursor])
	if cells := ansi.StringWidth(before); cells > width-1 {
		start := cursor
		for start > 0 && ansi.StringWidth(string(runes[start-1:cursor])) <= width-2 {
			start--
		}
		before = "…" + string(runes[start:cursor])
	}
	return ansi.Truncate(before+"█"+string(runes[cursor:]), width, "")
}

func (m model) addDialogView() string {
	width := max(12, min(84, m.width-10))
	spacious := m.height >= 24
	rows := []string{addLabelStyle.Foreground(addAccent).Render("Add torrent")}
	gap := func() {
		if spacious {
			rows = append(rows, "")
		}
	}
	input := func(index int, value, placeholder string) string {
		focused := m.addForm.field == index
		style := lipgloss.NewStyle().Foreground(addText).Background(addInputSurface).
			Border(lipgloss.NormalBorder()).BorderForeground(addSecondary).
			Width(width-2).Padding(0, 1)
		textWidth := width - 4
		if focused {
			style = style.BorderForeground(addAccent)
		}
		if value == "" {
			value = placeholder
			style = style.Foreground(addSecondary)
			if focused {
				value = "█" + value
			}
			value = ansi.Truncate(value, textWidth, "…")
		} else if focused {
			value = addInputWindow(value, m.addForm.cursors[index], textWidth)
		} else {
			value = ansi.Truncate(value, textWidth, "…")
		}
		return style.Render(value)
	}
	label := func(index int, text string) string {
		style := addLabelStyle
		if m.addForm.field == index {
			style = style.Foreground(addAccent)
		}
		return style.Render(text)
	}
	option := func(index int, enabled bool, text string) string {
		mark := "[ ] "
		if enabled {
			mark = "[x] "
		}
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(addText).Background(addSurface).Width(width)
		if m.addForm.field == index {
			prefix = "> "
			style = style.Foreground(addAccent).Background(addFocusSurface).Bold(true)
		}
		return style.Render(ansi.Truncate(prefix+mark+text, width, "…"))
	}
	pathHint := "Default download directory"
	if m.addForm.defaultPath != "" {
		pathHint = m.addForm.defaultPath
	}
	gap()
	rows = append(rows, label(addSource, "Source"), input(addSource, m.input, "Paste a magnet link or .torrent path"))
	if m.addForm.preview != "" {
		previewRows := strings.Split(ansi.Wrap(m.addForm.preview, width, ""), "\n")
		limit := 1
		if m.height >= 28 {
			limit = 3
		}
		for i, line := range previewRows[:min(limit, len(previewRows))] {
			if i == limit-1 && len(previewRows) > limit {
				line = ansi.Truncate(line, width-1, "") + "…"
			}
			rows = append(rows, addHintStyle.Render(line))
		}
	}
	if m.height >= 26 {
		rows = append(rows, "")
	}
	rows = append(rows, label(addPath, "Save to"), input(addPath, m.addForm.path, pathHint))
	gap()
	if spacious {
		rows = append(rows, addLabelStyle.Render("Download options"))
	}
	rows = append(rows, option(addSequential, m.addForm.sequential, "Sequential download"), option(addPaused, m.addForm.paused, "Start paused"))

	hint := ""
	switch m.addForm.field {
	case addPath:
		hint = "Leave blank for default. ~/ paths supported."
	case addSequential:
		hint = "Prioritize the beginning for early playback."
	case addPaused:
		hint = "Review files before starting the download."
	}
	hintStyle := addHintStyle
	if m.addForm.err != "" {
		hint = m.addForm.err
		hintStyle = errorStyle.Background(addSurface)
	}
	rows = append(rows, hintStyle.Render(ansi.Truncate(hint, width, "…")))
	gap()
	button := "  Add torrent  "
	buttonStyle := lipgloss.NewStyle().Foreground(addText).Background(addFocusSurface).Bold(true)
	if m.addForm.field == addSubmit && !m.addForm.cancel {
		buttonStyle = buttonStyle.Foreground(addInputSurface).Background(addAccent)
		button = "> Add torrent  "
	}
	if m.pending {
		button = "  Adding…  "
		buttonStyle = buttonStyle.Foreground(addSecondary).Background(addInputSurface)
	}
	cancelKey := dialogKeyStyle
	if m.addForm.field == addSubmit && m.addForm.cancel && !m.pending {
		cancelKey = cancelKey.Foreground(addInputSurface).Background(addAccent)
	}
	cancel := addHintStyle.Render("  ") + cancelKey.Render(" Esc ") + addHintStyle.Render(" cancel")
	rows = append(rows, buttonStyle.Render(button)+cancel)
	help := "↑/↓ field   Space toggle   Enter continue"
	if m.addForm.field <= addPath {
		help = "↑/↓ field   ←/→ cursor   Enter next"
	}
	if m.addForm.field == addSubmit {
		help = "←/→ choose   Enter confirm   ↑/↓ field"
	}
	if width < 40 {
		help = "↑/↓ field  Enter next"
		if m.addForm.field == addSequential || m.addForm.field == addPaused {
			help = "↑/↓ field  Space toggle"
		}
		if m.addForm.field == addSubmit {
			help = "←/→ choose  Enter confirm"
		}
	}
	if m.height >= 26 {
		rows = append(rows, "")
	}
	rows = append(rows, addHintStyle.Render(ansi.Truncate(help, width, "…")))
	for i, row := range rows {
		// Input borders span several lines; pad each line with the dialog surface.
		lines := strings.Split(row, "\n")
		for j, line := range lines {
			lines[j] = line + lipgloss.NewStyle().Background(addSurface).Render(strings.Repeat(" ", max(0, width-lipgloss.Width(line))))
		}
		rows[i] = strings.Join(lines, "\n")
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
