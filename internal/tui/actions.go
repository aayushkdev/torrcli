package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	domain "github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
)

type daemonClient interface {
	Add(context.Context, rpc.AddTorrentParams) (rpc.AddTorrentResult, error)
	List(context.Context) (rpc.ListTorrentsResult, error)
	Pause(context.Context, rpc.TorrentParams) (rpc.TorrentResult, error)
	Remove(context.Context, rpc.RemoveTorrentParams) error
	Resume(context.Context, rpc.TorrentParams) (rpc.TorrentResult, error)
}

func (m *model) toggleSelected() tea.Cmd {
	if m.pending {
		return nil
	}
	index := m.selectedIndex()
	if index < 0 {
		return nil
	}
	torrent := m.torrents[index]
	action := "pause"
	m.pending = true
	m.notice = "Pausing " + torrent.Name + "…"
	m.noticeErr = false
	if torrent.State == domain.TorrentStatePaused {
		action = "resume"
		m.notice = "Resuming " + torrent.Name + "…"
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		params := rpc.TorrentParams{ID: torrent.ID}
		if action == "pause" {
			result, err := m.daemon.Pause(ctx, params)
			return actionResultMsg{torrent: result.Torrent, action: action, err: err}
		}
		result, err := m.daemon.Resume(ctx, params)
		return actionResultMsg{torrent: result.Torrent, action: action, err: err}
	}
}

func (m model) updateDialog(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.dialog = dialogNone
		m.input = ""
	case tea.KeyEnter:
		switch m.dialog {
		case dialogAdd:
			source := strings.TrimSpace(m.input)
			if source == "" {
				return m, nil
			}
			m.dialog = dialogNone
			m.input = ""
			command := m.addTorrent(source)
			return m, command
		case dialogRemove:
			m.dialog = dialogNone
			command := m.removeSelected()
			return m, command
		}
	case tea.KeyBackspace:
		if m.dialog == dialogAdd && len(m.input) > 0 {
			runes := []rune(m.input)
			m.input = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		if m.dialog == dialogAdd {
			m.input += string(key.Runes)
		}
	case tea.KeySpace:
		if m.dialog == dialogAdd {
			m.input += " "
		}
	}
	return m, nil
}

func (m *model) addTorrent(source string) tea.Cmd {
	m.pending = true
	m.notice = "Adding torrent…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		result, err := m.daemon.Add(ctx, rpc.AddTorrentParams{Source: source})
		return actionResultMsg{torrent: result.Torrent, action: "add", err: err}
	}
}

func (m *model) removeSelected() tea.Cmd {
	index := m.selectedIndex()
	if index < 0 || m.pending {
		return nil
	}
	torrent := m.torrents[index]
	m.pending = true
	m.notice = "Removing " + torrent.Name + "…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		err := m.daemon.Remove(ctx, rpc.RemoveTorrentParams{ID: torrent.ID})
		return actionResultMsg{removedID: torrent.ID, action: "remove", err: err}
	}
}

func (m *model) handleActionResult(message actionResultMsg) tea.Cmd {
	m.pending = false
	if message.err != nil {
		return m.setNotice("Could not "+message.action+": "+message.err.Error(), true)
	}
	switch message.action {
	case "pause", "resume":
		m.replaceTorrent(message.torrent)
		return m.setNotice(strings.ToUpper(message.action[:1])+message.action[1:], false)
	case "add":
		m.upsertTorrent(message.torrent)
		m.ensureSelection()
		return m.setNotice("Added "+message.torrent.Name, false)
	case "remove":
		m.removeTorrent(message.removedID)
		return m.setNotice("Removed torrent", false)
	}
	return nil
}

func (m *model) setNotice(notice string, isError bool) tea.Cmd {
	m.noticeID++
	m.notice = notice
	m.noticeErr = isError
	id := m.noticeID
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg { return noticeClearedMsg(id) })
}

func (m *model) replaceTorrent(updated domain.TorrentSnapshot) {
	for index, torrent := range m.torrents {
		if torrent.ID == updated.ID {
			m.torrents[index] = updated
			return
		}
	}
}

func (m *model) upsertTorrent(updated domain.TorrentSnapshot) {
	for index, torrent := range m.torrents {
		if torrent.ID == updated.ID {
			m.torrents[index] = updated
			return
		}
	}
	m.torrents = append(m.torrents, updated)
}

func (m *model) removeTorrent(id domain.TorrentID) {
	for index, torrent := range m.torrents {
		if torrent.ID != id {
			continue
		}
		m.torrents = append(m.torrents[:index], m.torrents[index+1:]...)
		m.ensureSelection()
		return
	}
}
