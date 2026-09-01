package state_test

import (
	"os"
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
	if len(session.Order) != 0 || len(session.Torrents) != 0 {
		t.Fatalf("unexpected default session: %#v", session)
	}
}

func TestLoadSessionRecoversFromBackup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session.json")
	if _, err := state.LoadOrCreateSession(path); err != nil {
		t.Fatalf("create session: %v", err)
	}

	session := model.DefaultSession()
	session.Order = []model.TorrentID{"torrent-a"}
	session.Torrents["torrent-a"] = model.TorrentRecord{
		Source:       "magnet:?xt=urn:btih:abc",
		SavePath:     "/downloads",
		DesiredState: model.TorrentStatePaused,
	}
	if err := state.SaveSession(path, session); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("corrupt session: %v", err)
	}

	recovered, err := state.LoadOrCreateSession(path)
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if len(recovered.Order) != 0 {
		t.Fatalf("recovered order = %#v, want backup session", recovered.Order)
	}
}
