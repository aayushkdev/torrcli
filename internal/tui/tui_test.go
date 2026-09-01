package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelHandlesWindowSize(t *testing.T) {
	updated, _ := model{}.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	actual := updated.(model)
	if actual.width != 80 || actual.height != 24 {
		t.Fatalf("model size = %dx%d", actual.width, actual.height)
	}
}
