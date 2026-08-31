// Package platform provides operating-system integration helpers.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Paths identifies all application-owned directories and files.
type Paths struct {
	ConfigDir   string
	StateDir    string
	RuntimeDir  string
	ConfigFile  string
	SessionFile string
	SocketPath  string
	LockFile    string
}

// DefaultPaths returns platform-appropriate locations for torrcli data.
func DefaultPaths() (Paths, error) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("get user config directory: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("get user home directory: %w", err)
	}

	configDir := filepath.Join(configRoot, "torrcli")
	stateDir := defaultStateDir(configDir, home)
	runtimeDir := defaultRuntimeDir(stateDir)

	return newPaths(configDir, stateDir, runtimeDir), nil
}

// WithOverrides returns paths with non-empty directory overrides applied.
func (p Paths) WithOverrides(configDir, stateDir, runtimeDir string) Paths {
	if configDir != "" {
		p.ConfigDir = configDir
	}
	if stateDir != "" {
		p.StateDir = stateDir
	}
	if runtimeDir != "" {
		p.RuntimeDir = runtimeDir
	}
	return newPaths(p.ConfigDir, p.StateDir, p.RuntimeDir)
}

func newPaths(configDir, stateDir, runtimeDir string) Paths {
	return Paths{
		ConfigDir:   configDir,
		StateDir:    stateDir,
		RuntimeDir:  runtimeDir,
		ConfigFile:  filepath.Join(configDir, "config.json"),
		SessionFile: filepath.Join(stateDir, "session.json"),
		SocketPath:  filepath.Join(runtimeDir, "torrd.sock"),
		LockFile:    filepath.Join(runtimeDir, "torrd.lock"),
	}
}

func defaultStateDir(configDir, home string) string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return configDir
	}
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "torrcli")
	}
	return filepath.Join(home, ".local", "state", "torrcli")
}

func defaultRuntimeDir(stateDir string) string {
	if runtime.GOOS != "windows" {
		if runtimeHome := os.Getenv("XDG_RUNTIME_DIR"); runtimeHome != "" {
			return filepath.Join(runtimeHome, "torrcli")
		}
	}
	return filepath.Join(stateDir, "runtime")
}
