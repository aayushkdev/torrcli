package tui

import (
	"context"
	"testing"

	domain "github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	tea "github.com/charmbracelet/bubbletea"
)

type removeClient struct {
	daemonClient
	params rpc.RemoveTorrentParams
}

func (c *removeClient) Remove(_ context.Context, p rpc.RemoveTorrentParams) error {
	c.params = p
	return nil
}

func TestRemoveDialogSubmitsDeleteChoice(t *testing.T) {
	for _, deleteData := range []bool{false, true} {
		client := &removeClient{}
		m := model{ctx: context.Background(), daemon: client, dialog: dialogRemove, selectedID: "one", removeDeleteData: deleteData, torrents: []domain.TorrentSnapshot{{ID: "one"}}}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("remove not submitted")
		}
		cmd()
		if client.params.ID != "one" || client.params.DeleteData != deleteData {
			t.Fatalf("remove params: %+v", client.params)
		}
	}
}
