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
	Info(context.Context) (rpc.DaemonInfo, error)
	Add(context.Context, rpc.AddTorrentParams) (rpc.AddTorrentResult, error)
	List(context.Context) (rpc.ListTorrentsResult, error)
	Details(context.Context, rpc.TorrentParams) (rpc.TorrentDetailsResult, error)
	Pause(context.Context, rpc.TorrentParams) (rpc.TorrentResult, error)
	Remove(context.Context, rpc.RemoveTorrentParams) error
	Resume(context.Context, rpc.TorrentParams) (rpc.TorrentResult, error)
	Verify(context.Context, rpc.TorrentParams) (rpc.TorrentResult, error)
	FindPeers(context.Context, rpc.TorrentParams) error
	SetSequential(context.Context, rpc.SetSequentialParams) (rpc.TorrentResult, error)
	SetFilePriority(context.Context, rpc.SetFilePriorityParams) (rpc.TorrentResult, error)
	Move(context.Context, rpc.MoveTorrentParams) (rpc.ListTorrentsResult, error)
}

type torrentAction string

const (
	actionToggle     torrentAction = "toggle"
	actionVerify     torrentAction = "verify"
	actionFindPeers  torrentAction = "find peers"
	actionSequential torrentAction = "sequential"
	actionRemove     torrentAction = "remove"
)

func (m model) torrentActions() []torrentAction {

	return []torrentAction{actionToggle, actionVerify, actionFindPeers, actionSequential, actionRemove}
}

func (m model) actionLabel(action torrentAction) string {
	if action == actionToggle {
		if index := m.selectedIndex(); index >= 0 && m.torrents[index].State == domain.TorrentStatePaused {
			return "Resume"
		}
		return "Pause"
	}
	switch action {
	case actionSequential:
		if index := m.selectedIndex(); index >= 0 && m.torrents[index].Sequential {
			return "Disable sequential download"
		}
		return "Enable sequential download"
	case actionVerify:
		return "Force recheck"
	case actionFindPeers:
		return "Find peers now"
	default:
		return "Remove torrent"
	}
}

func (m model) actionEnabled(action torrentAction) bool {
	if action != actionFindPeers {
		return true
	}
	index := m.selectedIndex()
	return index >= 0 && m.torrents[index].State != domain.TorrentStateFetchingMetadata
}

func (m *model) verifySelected() tea.Cmd {
	if m.pending || m.selectedID == "" {
		return nil
	}
	m.pending = true
	m.notice = "Checking torrent data…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Minute)
		defer cancel()
		result, err := m.daemon.Verify(ctx, rpc.TorrentParams{ID: m.selectedID})
		return actionResultMsg{torrent: result.Torrent, action: "verify", err: err}
	}
}

func (m *model) findPeersSelected() tea.Cmd {
	if m.pending || m.selectedID == "" {
		return nil
	}
	m.pending = true
	m.notice = "Looking for peers…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		err := m.daemon.FindPeers(ctx, rpc.TorrentParams{ID: m.selectedID})
		return actionResultMsg{action: "find peers", err: err}
	}
}

func (m *model) moveSelected(offset int) tea.Cmd {
	if m.pending || m.selectedID == "" {
		return nil
	}
	m.pending = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		result, err := m.daemon.Move(ctx, rpc.MoveTorrentParams{ID: m.selectedID, Offset: offset})
		return torrentsMovedMsg{torrents: result.Torrents, err: err}
	}
}

func (m *model) setSelectedFilePriority() tea.Cmd {
	if m.pending || m.selectedFile < 0 || m.selectedFile >= len(m.details.Files) {
		return nil
	}
	file := m.details.Files[m.selectedFile]
	priority := nextPriority(file.Priority)
	m.pending = true
	m.notice = "Updating " + file.Path + "…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		result, err := m.daemon.SetFilePriority(ctx, rpc.SetFilePriorityParams{ID: m.details.Torrent.ID, FileIndex: file.Index, Priority: priority})
		return filePriorityResultMsg{torrent: result.Torrent, file: file.Index, priority: priority, err: err}
	}
}

func nextPriority(priority domain.FilePriority) domain.FilePriority {
	switch priority {
	case domain.FilePrioritySkip:
		return domain.FilePriorityNormal
	case domain.FilePriorityNormal:
		return domain.FilePriorityHigh
	default:
		return domain.FilePrioritySkip
	}
}

func (m *model) loadDetails() tea.Cmd {
	index := m.selectedIndex()
	if index < 0 {
		return nil
	}
	id := m.torrents[index].ID
	m.detailsLoading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		result, err := m.daemon.Details(ctx, rpc.TorrentParams{ID: id})
		return detailsLoadedMsg{id: id, details: result.Details, err: err}
	}
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
	if m.dialog == dialogAdd {
		return m.updateAddDialog(key)
	}
	switch key.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.dialog = dialogNone
		m.input = ""
	case tea.KeyEnter:
		switch m.dialog {
		case dialogActions:
			actions := m.torrentActions()
			if len(actions) == 0 {
				return m, nil
			}
			action := actions[m.actionIndex]
			if !m.actionEnabled(action) {
				return m, nil
			}
			m.dialog = dialogNone
			switch action {
			case actionToggle:
				return m, m.toggleSelected()
			case actionSequential:
				command := m.toggleSequential()
				return m, command
			case actionVerify:
				return m, m.verifySelected()
			case actionFindPeers:
				return m, m.findPeersSelected()
			case actionRemove:
				m.dialog = dialogRemove
				m.removeCancel = false
				m.removeDeleteData = false
				m.removeOption = false
			}
			return m, nil
		case dialogRemove:
			if m.removeOption {
				m.removeDeleteData = !m.removeDeleteData
				return m, nil
			}
			m.dialog = dialogNone
			if m.removeCancel {
				return m, nil
			}
			command := m.removeSelected()
			return m, command
		}
	}

	if m.dialog == dialogRemove {
		switch key.Type {
		case tea.KeyLeft, tea.KeyRight:
			m.removeOption = false
			m.removeCancel = !m.removeCancel
		case tea.KeyUp, tea.KeyDown:
			m.removeOption = !m.removeOption
		case tea.KeySpace:
			if m.removeOption {
				m.removeDeleteData = !m.removeDeleteData
			}
		case tea.KeyTab, tea.KeyShiftTab:
			if m.removeOption {
				m.removeOption = false
				m.removeCancel = false
			} else if !m.removeCancel {
				m.removeCancel = true
			} else {
				m.removeOption = true
			}
		}
	}
	if m.dialog == dialogActions {
		switch key.String() {
		case "up", "k":
			m.actionIndex = max(0, m.actionIndex-1)
		case "down", "j":
			m.actionIndex = min(len(m.torrentActions())-1, m.actionIndex+1)
		}
	}
	return m, nil
}

func (m *model) addTorrent(params rpc.AddTorrentParams) tea.Cmd {
	m.pending = true
	m.notice = "Adding torrent…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		result, err := m.daemon.Add(ctx, params)
		return actionResultMsg{torrent: result.Torrent, action: "add", err: err}
	}
}

func (m *model) removeSelected() tea.Cmd {
	index := m.selectedIndex()
	if index < 0 || m.pending {
		return nil
	}
	torrent := m.torrents[index]
	deleteData := m.removeDeleteData
	m.pending = true
	m.notice = "Removing " + torrent.Name + "…"
	m.noticeErr = false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		err := m.daemon.Remove(ctx, rpc.RemoveTorrentParams{ID: torrent.ID, DeleteData: deleteData})
		return actionResultMsg{removedID: torrent.ID, action: "remove", err: err}
	}
}

func (m *model) handleActionResult(message actionResultMsg) tea.Cmd {
	m.pending = false
	if message.err != nil {
		if message.action == "add" {
			m.addForm.err = message.err.Error()
		}
		return m.setNotice("Could not "+message.action+": "+message.err.Error(), true)
	}
	switch message.action {
	case "pause", "resume":
		m.replaceTorrent(message.torrent)
		return m.setNotice(strings.ToUpper(message.action[:1])+message.action[1:], false)
	case "sequential":
		m.replaceTorrent(message.torrent)
		if message.torrent.Sequential {
			return m.setNotice("Sequential download enabled", false)
		}
		return m.setNotice("Sequential download disabled", false)
	case "verify":
		m.replaceTorrent(message.torrent)
		return m.setNotice("Verification complete", false)
	case "find peers":
		return m.setNotice("Searching DHT for peers", false)
	case "add":
		m.dialog = dialogNone
		m.input = ""
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

func (m *model) toggleSequential() tea.Cmd {
	index := m.selectedIndex()
	if m.pending || index < 0 {
		return nil
	}
	t := m.torrents[index]
	m.pending = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Second)
		defer cancel()
		result, err := m.daemon.SetSequential(ctx, rpc.SetSequentialParams{ID: t.ID, Enabled: !t.Sequential})
		return actionResultMsg{action: "sequential", torrent: result.Torrent, err: err}
	}
}
