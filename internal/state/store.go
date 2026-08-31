// Package state manages file-based configuration and session persistence.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/aayush/torrcli/internal/model"
)

// LoadOrCreateConfig loads configPath or writes defaults when it is absent.
func LoadOrCreateConfig(configPath, downloadDirectory string) (model.Config, error) {
	var config model.Config
	err := readJSON(configPath, &config)
	if errors.Is(err, fs.ErrNotExist) {
		config = model.DefaultConfig(downloadDirectory)
		if err := SaveConfig(configPath, config); err != nil {
			return model.Config{}, err
		}
		return config, nil
	}
	if err != nil {
		return model.Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return model.Config{}, err
	}
	return config, nil
}

// SaveConfig atomically persists config.
func SaveConfig(configPath string, config model.Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}
	if err := writeJSON(configPath, config); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// LoadOrCreateSession loads sessionPath or writes an empty session when absent.
func LoadOrCreateSession(sessionPath string) (model.Session, error) {
	var session model.Session
	err := readJSON(sessionPath, &session)
	if errors.Is(err, fs.ErrNotExist) {
		session = model.DefaultSession()
		if err := SaveSession(sessionPath, session); err != nil {
			return model.Session{}, err
		}
		return session, nil
	}
	if err != nil {
		return model.Session{}, fmt.Errorf("read session: %w", err)
	}
	if err := validateSession(session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

// SaveSession atomically persists session.
func SaveSession(sessionPath string, session model.Session) error {
	if err := validateSession(session); err != nil {
		return err
	}
	if err := writeJSON(sessionPath, session); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func readJSON(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func writeJSON(path string, value any) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	backupPath := path + ".bak"
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backupPath); err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}

	directoryFile, err := os.Open(directory)
	if err == nil {
		defer directoryFile.Close()
		_ = directoryFile.Sync()
	}
	return nil
}

func validateConfig(config model.Config) error {
	if config.Version != model.StateVersion {
		return fmt.Errorf("unsupported config version %d", config.Version)
	}
	if config.DownloadDirectory == "" {
		return errors.New("download directory is required")
	}
	if config.MaxActiveDownloads < 0 || config.MaxActiveSeeds < 0 {
		return errors.New("active torrent limits cannot be negative")
	}
	if config.DownloadLimit < 0 || config.UploadLimit < 0 {
		return errors.New("transfer limits cannot be negative")
	}
	if config.ListenPort < 0 || config.ListenPort > 65535 {
		return errors.New("listen port must be between 0 and 65535")
	}
	return nil
}

func validateSession(session model.Session) error {
	if session.Version != model.StateVersion {
		return fmt.Errorf("unsupported session version %d", session.Version)
	}
	if session.Order == nil {
		return errors.New("session order is required")
	}
	if session.Torrents == nil {
		return errors.New("session torrents are required")
	}
	return nil
}
