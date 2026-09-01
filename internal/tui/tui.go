package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aayush/torrcli/internal/client"
)

func Run(ctx context.Context, daemon *client.Client) error {
	_, err := tea.NewProgram(model{ctx: ctx, daemon: daemon}, tea.WithAltScreen()).Run()
	return err
}
