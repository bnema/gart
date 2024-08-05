package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// LoadConfig loads the config.toml file and returns a map of strings
func LoadConfig(configPath string) (map[string]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		fmt.Printf("Failed to read config file: %v\n", err)
		return nil, err
	}
	var config map[string]string
	err = toml.Unmarshal(data, &config)

	return config, err
}

func SaveConfig(ConfigFilePath string, dotfiles map[string]string) error {
	data, err := toml.Marshal(dotfiles)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return err
	}

	err = os.WriteFile(ConfigFilePath, data, 0664)

	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}

	return nil
}
