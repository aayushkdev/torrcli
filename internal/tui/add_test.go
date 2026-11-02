package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aayush/torrcli/internal/rpc"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addClient struct {
	daemonClient
	params rpc.AddTorrentParams
	err    error
}

func (c *addClient) Add(_ context.Context, params rpc.AddTorrentParams) (rpc.AddTorrentResult, error) {
	c.params = params
	return rpc.AddTorrentResult{}, c.err
}

func TestAddDialogArrowNavigationAndSubmission(t *testing.T) {
	client := &addClient{}
	m := model{ctx: context.Background(), daemon: client, dialog: dialogAdd}
	key := func(k tea.KeyMsg) tea.Cmd {
		updated, cmd := m.Update(k)
		m = updated.(model)
		return cmd
	}
	key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("movie.torrent")})
	key(tea.KeyMsg{Type: tea.KeyDown})
	directory := filepath.Join(t.TempDir(), "my movies")
	key(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(directory)})
	key(tea.KeyMsg{Type: tea.KeyDown})
	key(tea.KeyMsg{Type: tea.KeySpace})
	key(tea.KeyMsg{Type: tea.KeyDown})
	key(tea.KeyMsg{Type: tea.KeyEnter})
	key(tea.KeyMsg{Type: tea.KeyDown})
	cmd := key(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.pending {
		t.Fatal("submission did not start")
	}
	cmd()
	if client.params.SavePath != directory || !client.params.Sequential || !client.params.Paused || !filepath.IsAbs(client.params.Source) {
		t.Fatalf("add params: %+v", client.params)
	}
}

func TestAddDialogEditsAtCursorAndRetainsFailedInput(t *testing.T) {
	client := &addClient{err: errors.New("directory is not writable")}
	m := model{ctx: context.Background(), daemon: client, dialog: dialogAdd}
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("ab.torrent")},
		{Type: tea.KeyHome}, {Type: tea.KeyRight},
		{Type: tea.KeyRunes, Runes: []rune("é")},
	} {
		updated, _ := m.Update(key)
		m = updated.(model)
	}
	if m.input != "aéb.torrent" {
		t.Fatalf("edited source: %s", m.input)
	}
	m.addForm.field = addSubmit
	m.addForm.path = "/downloads"
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if m.dialog != dialogAdd || m.input != "aéb.torrent" || m.addForm.path != "/downloads" || m.addForm.err == "" || m.pending {
		t.Fatalf("form lost on error: %+v", m.addForm)
	}
}

func TestAddDialogLayout(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		m := model{width: width, height: 24, dialog: dialogAdd, input: strings.Repeat("界", 100), addForm: addForm{defaultPath: "/downloads", preview: strings.Repeat("movie", 30)}}
		rendered := addDialogStyle.Render(m.addDialogView())
		if lipgloss.Width(rendered) > width || lipgloss.Height(rendered) > m.height {
			t.Fatalf("dialog overflows %dx%d: %dx%d", width, m.height, lipgloss.Width(rendered), lipgloss.Height(rendered))
		}
		for _, label := range []string{"Save to", "Sequential download", "Start paused", "Add torrent"} {
			if !strings.Contains(rendered, label) {
				t.Fatalf("missing %q", label)
			}
		}
	}
}

func TestTypedTextRemainsVisibleInBothAddFields(t *testing.T) {
	for _, field := range []int{addSource, addPath} {
		m := model{width: 80, height: 24, dialog: dialogAdd, addForm: addForm{field: field}}
		typed := ""
		for _, r := range "movie é.mkv" {
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(model)
			typed += string(r)
			if !strings.Contains(m.addDialogView(), typed+"█") {
				t.Fatalf("field %d hides typed text %q", field, typed)
			}
		}
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		m = updated.(model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(model)
		if !strings.Contains(m.addDialogView(), "movie é.m█v") {
			t.Fatalf("field %d hides cursor edit", field)
		}
	}
}

func TestAddInputWindowScrollsAroundCursor(t *testing.T) {
	for _, value := range []string{"abc", strings.Repeat("a", 100) + "END", strings.Repeat("界", 100) + "END"} {
		end := addInputWindow(value, len([]rune(value)), 20)
		if lipgloss.Width(end) > 20 || !strings.HasSuffix(end, "█") {
			t.Fatalf("end cursor hidden: %q", end)
		}
		if !strings.Contains(end, "END") && len(value) > 100 {
			t.Fatalf("end of value hidden: %q", end)
		}
		start := addInputWindow(value, 0, 20)
		if !strings.HasPrefix(start, "█") || lipgloss.Width(start) > 20 {
			t.Fatalf("start cursor hidden: %q", start)
		}
	}
}

func TestPastedKeyNamesStayText(t *testing.T) {
	for _, text := range []string{"up", "down", "enter", "tab", "esc"} {
		m := model{dialog: dialogAdd}
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
		m = updated.(model)
		if m.input != text || m.addForm.field != addSource || m.dialog != dialogAdd {
			t.Fatalf("text %q acted as a key", text)
		}
	}
}

func TestControlCClearsThenClosesThenQuits(t *testing.T) {
	for _, field := range []int{addSource, addPath} {
		m := model{dialog: dialogAdd, focus: focusDetails, input: "magnet:?xt=example", addForm: addForm{field: field, path: "/movies", cursors: [2]int{18, 7}, preview: "old metadata", err: "old error"}}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m = updated.(model)
		if cmd != nil || m.dialog != dialogAdd || m.addForm.cursors[field] != 0 || m.addForm.err != "" {
			t.Fatal("first Ctrl+C did not clear the active input")
		}
		if field == addSource && (m.input != "" || m.addForm.preview != "") {
			t.Fatal("source not cleared")
		}
		if field == addPath && (m.addForm.path != "" || m.input == "") {
			t.Fatal("path clear changed source")
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m = updated.(model)
		if cmd != nil || m.dialog != dialogNone {
			t.Fatal("second Ctrl+C did not close dialog")
		}
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatal("third Ctrl+C did not quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("expected quit from details focus")
		}
	}
}

func TestAddDialogKeepsPreviewAboveOptionsAndFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 24}, {80, 24}, {100, 26}, {120, 28}, {160, 40}} {
		m := model{width: size[0], height: size[1], dialog: dialogAdd, addForm: addForm{preview: "Example movie with a long title · Size and files available after metadata loads"}}
		view := m.addDialogView()
		if strings.Index(view, "Example movie") < strings.Index(view, "Source") {
			t.Fatal("preview must follow the source field")
		}
		rendered := addDialogStyle.Render(view)
		if lipgloss.Height(rendered) > m.height || lipgloss.Width(rendered) > m.width {
			t.Fatalf("dialog %dx%d overflows terminal %dx%d", lipgloss.Width(rendered), lipgloss.Height(rendered), m.width, m.height)
		}
	}
}

func TestAddDialogCanSelectCancelWithArrows(t *testing.T) {
	for _, arrow := range []tea.KeyType{tea.KeyLeft, tea.KeyRight} {
		m := model{dialog: dialogAdd, input: "example.torrent", addForm: addForm{field: addSubmit}}
		updated, _ := m.Update(tea.KeyMsg{Type: arrow})
		m = updated.(model)
		if !m.addForm.cancel {
			t.Fatal("arrow did not select cancel")
		}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(model)
		if cmd != nil || m.pending || m.dialog != dialogNone {
			t.Fatal("confirming cancel attempted to add")
		}
	}
}
