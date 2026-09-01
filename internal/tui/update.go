package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadTorrents(), refresh())
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		if m.dialog != dialogNone {
			return m.updateDialog(message)
		}
		if m.showDetails {
			switch message.String() {
			case "esc", "q":
				m.showDetails = false
				m.detailsErr = nil
			case "up", "k":
				m.selectPreviousFile()
			case "down", "j":
				m.selectNextFile()
			case " ":
				command := m.setSelectedFilePriority()
				return m, command
			}
			return m, nil
		}
		switch message.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.selectPrevious()
		case "down", "j":
			m.selectNext()
		case " ":
			command := m.toggleSelected()
			return m, command
		case "a":
			m.dialog = dialogAdd
			m.input = ""
		case "d":
			if m.selectedIndex() >= 0 && !m.pending {
				m.dialog = dialogRemove
			}
		case "enter":
			command := m.loadDetails()
			return m, command
		case "shift+up":
			command := m.moveSelected(-1)
			return m, command
		case "shift+down":
			command := m.moveSelected(1)
			return m, command
		}
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	case torrentsLoadedMsg:
		m.loading = false
		if message.err != nil {
			m.loadError = message.err
			return m, nil
		}
		m.torrents = message.torrents
		m.loadError = nil
		m.ensureSelection()
	case refreshMsg:
		return m, tea.Batch(m.loadTorrents(), refresh())
	case actionResultMsg:
		command := m.handleActionResult(message)
		return m, command
	case noticeClearedMsg:
		if int(message) == m.noticeID {
			m.notice = ""
			m.noticeErr = false
		}
	case detailsLoadedMsg:
		m.pending = false
		if message.err != nil {
			m.detailsErr = message.err
			m.showDetails = true
			return m, nil
		}
		m.details = message.details
		m.selectedFile = 0
		m.detailsErr = nil
		m.showDetails = true
	case filePriorityResultMsg:
		m.pending = false
		if message.err != nil {
			command := m.setNotice("Could not update priority: "+message.err.Error(), true)
			return m, command
		}
		m.details.Torrent = message.torrent
		m.replaceTorrent(message.torrent)
		for index, file := range m.details.Files {
			if file.Index == message.file {
				m.details.Files[index].Priority = message.priority
				break
			}
		}
		command := m.setNotice("File priority set to "+string(message.priority), false)
		return m, command
	case torrentsMovedMsg:
		m.pending = false
		if message.err != nil {
			command := m.setNotice("Could not move torrent: "+message.err.Error(), true)
			return m, command
		}
		m.torrents = message.torrents
		m.ensureSelection()
		command := m.setNotice("Torrent moved", false)
		return m, command
	}
	return m, nil
}

func (m *model) selectPreviousFile() {
	if m.selectedFile > 0 {
		m.selectedFile--
	}
}

func (m *model) selectNextFile() {
	if m.selectedFile < len(m.details.Files)-1 {
		m.selectedFile++
	}
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
