package system

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetOSConfigPath() (string, error) {
	var configPath string

	switch runtime.GOOS {
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "gart")
	case "linux", "darwin":
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "gart")
	default:
		configPath = ".config"
	}

	// Return an error if we cannot determine the config path
	if configPath == "" {
		return "", os.ErrNotExist
	}

	// Create the config directory if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err := os.MkdirAll(configPath, 0755)
		if err != nil {
			return "", err
		}
	}

	return configPath, nil
}
