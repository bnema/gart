package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetOSConfigPath() string {
	var configPath string

	switch runtime.GOOS {
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "Gart")
	case "linux", "darwin":
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "gart")
	default:
		configPath = ".config"
	}

	return configPath
}
