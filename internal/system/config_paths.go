package system

import (
	"os"
	"path/filepath"
)

func GetConfigPaths() (string, string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", "", err
    }

    xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
    if xdgConfigHome == "" {
        xdgConfigHome = filepath.Join(homeDir, ".config")
    }

    gartConfigDir := filepath.Join(xdgConfigHome, "gart")
    configPath := filepath.Join(gartConfigDir, "config.toml")

    return gartConfigDir, configPath, nil
}
