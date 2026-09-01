package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(1, 2)
)

type model struct {
	width  int
	height int
}

func Run() error {
	_, err := tea.NewProgram(model{}, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("torrcli"),
		mutedStyle.Render("Connected to torrd"),
		"",
		"Torrent table is coming next.",
	)
	help := mutedStyle.Render("q quit")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Left, panelStyle.Render(content), fmt.Sprintf("\n%s", help)))
}
