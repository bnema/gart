package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

func LoadConfig(configPath string, dotfiles map[string]string) {

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	for key, value := range config {
		if path, ok := value.(string); ok {
			dotfiles[key] = path
		}
	}
}

func SaveConfig(dotfiles map[string]string) {
	data, err := toml.Marshal(dotfiles)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}

	err = os.WriteFile("config.toml", data, 0644)
	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}
