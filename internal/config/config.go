package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// Config represents the structure of the entire configuration file
type Config struct {
	Dotfiles map[string]string `toml:"dotfiles"`
}

// Dotfile represents the structure of one dotfile entry
// example: kitty = "/home/user/.config/kitty"
type Dotfile struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

func LoadConfig(configPath string) (Config, error) {
	var config Config
	_, err := toml.LoadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("error loading config file: %w", err)
	}

	return config, nil
}

func SaveConfig(configFilePath string, config Config) error {
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	return os.WriteFile(configFilePath, data, 0664)
}
