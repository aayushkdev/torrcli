package state_test

import (
	"path/filepath"
	"testing"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/state"
)

func TestLoadOrCreateConfigAndSession(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	sessionPath := filepath.Join(directory, "session.json")

	config, err := state.LoadOrCreateConfig(configPath, "/downloads")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.DownloadDirectory != "/downloads" {
		t.Fatalf("download directory = %q, want %q", config.DownloadDirectory, "/downloads")
	}

	config.MaxActiveDownloads = 7
	if err := state.SaveConfig(configPath, config); err != nil {
		t.Fatalf("save config: %v", err)
	}
	reloadedConfig, err := state.LoadOrCreateConfig(configPath, "/ignored")
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloadedConfig.MaxActiveDownloads != 7 {
		t.Fatalf("max active downloads = %d, want 7", reloadedConfig.MaxActiveDownloads)
	}

	session, err := state.LoadOrCreateSession(sessionPath)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.Version != model.StateVersion || len(session.Order) != 0 || len(session.Torrents) != 0 {
		t.Fatalf("unexpected default session: %#v", session)
	}
}
