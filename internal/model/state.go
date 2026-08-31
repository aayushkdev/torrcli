package model

import "time"

const StateVersion = 1

type Config struct {
	Version            int    `json:"version"`
	DownloadDirectory  string `json:"download_directory"`
	MaxActiveDownloads int    `json:"max_active_downloads"`
	MaxActiveSeeds     int    `json:"max_active_seeds"`
	DownloadLimit      int64  `json:"download_limit"`
	UploadLimit        int64  `json:"upload_limit"`
	ListenPort         int    `json:"listen_port"`
}

func DefaultConfig(downloadDirectory string) Config {
	return Config{
		Version:            StateVersion,
		DownloadDirectory:  downloadDirectory,
		MaxActiveDownloads: 3,
		MaxActiveSeeds:     5,
	}
}

type Session struct {
	Version  int                      `json:"version"`
	Order    []string                 `json:"order"`
	Torrents map[string]TorrentRecord `json:"torrents"`
}

type TorrentRecord struct {
	Source         string            `json:"source"`
	Name           string            `json:"name"`
	SavePath       string            `json:"save_path"`
	DesiredState   string            `json:"desired_state"`
	AddedAt        time.Time         `json:"added_at"`
	FilePriorities map[string]string `json:"file_priorities,omitempty"`
}

func DefaultSession() Session {
	return Session{
		Version:  StateVersion,
		Order:    []string{},
		Torrents: map[string]TorrentRecord{},
	}
}
