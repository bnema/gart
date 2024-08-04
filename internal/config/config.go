package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

func LoadConfig(configPath string, dotfiles map[string]string) (map[string]string, error) {

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return dotfiles, nil
		}
		fmt.Printf("Failed to read config file: %v\n", err)
		return nil, err
	}

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return nil, err
	}

	for key, value := range config {
		if path, ok := value.(string); ok {
			dotfiles[key] = path
		}
	}

	return dotfiles, nil
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
