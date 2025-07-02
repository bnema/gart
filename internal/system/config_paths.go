package system

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigPaths returns the config directory and config file path for gart
func GetConfigPaths() (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	var gartConfigDir string

	switch runtime.GOOS {
	case "windows":
		// Use APPDATA for config on Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		gartConfigDir = filepath.Join(appData, "gart")

	case "darwin":
		// Use ~/Library/Preferences on macOS
		gartConfigDir = filepath.Join(homeDir, "Library", "Preferences", "gart")

	default:
		// Linux and other Unix-like systems - use XDG spec
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}
		gartConfigDir = filepath.Join(xdgConfigHome, "gart")
	}

	configPath := filepath.Join(gartConfigDir, "config.toml")
	return gartConfigDir, configPath, nil
}

// GetDataPaths returns the data directory path for gart storage
func GetDataPaths() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var gartDataDir string

	switch runtime.GOOS {
	case "windows":
		// Use LOCALAPPDATA for data on Windows
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		gartDataDir = filepath.Join(localAppData, "gart")

	case "darwin":
		// Use ~/Library/Application Support on macOS
		gartDataDir = filepath.Join(homeDir, "Library", "Application Support", "gart")

	default:
		// Linux and other Unix-like systems - use XDG spec
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}
		gartDataDir = filepath.Join(xdgDataHome, "gart")
	}

	return gartDataDir, nil
}
