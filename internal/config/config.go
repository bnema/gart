package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/gart/internal/system"
	"github.com/pelletier/go-toml"
)

// Config represents the structure of the entire configuration file
type Config struct {
	Settings SettingsConfig    `toml:"settings"`
	Dotfiles map[string]string `toml:"dotfiles"`
}

// SettingsConfig represents the general settings of the application
type SettingsConfig struct {
	StoragePath   string    `toml:"storage_path"`
	GitVersioning bool      `toml:"git_versioning"`
	Git           GitConfig `toml:"git"`
}

// GitConfig represents the structure of the git configuration
type GitConfig struct {
	Branch              string `toml:"branch"`
	CommitMessageFormat string `toml:"commit_message_format"`
}

// LoadConfig loads the configuration from the file
func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &config, nil
}

// AddDotfileToConfig adds a new dotfile to the config file
func AddDotfileToConfig(configPath string, name, path string) error {
	tree, err := toml.LoadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			tree, _ = toml.Load("")
		} else {
			return fmt.Errorf("error loading config file: %w", err)
		}
	}

	dotfilesTree, ok := tree.Get("dotfiles").(*toml.Tree)
	if !ok {
		dotfilesTree, _ = toml.Load("")
		tree.Set("dotfiles", dotfilesTree)
	}

	dotfilesTree.Set(name, path)

	f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	err = encoder.Encode(tree)
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	return nil
}

// SaveConfig saves the configuration to the file
func SaveConfig(configPath string, config *Config) error {
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0664)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig(configPath string) (*Config, error) {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error getting user home directory: %w", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	gartConfigDir := filepath.Join(xdgConfigHome, "gart")

	configPath = filepath.Join(gartConfigDir, "config.toml")

	branchName, err := system.GetHostname()
	if err != nil {
		branchName = "main"
	}

	config := &Config{
		Settings: SettingsConfig{
			StoragePath:   filepath.Join(gartConfigDir, ".store"),
			GitVersioning: false,
			Git: GitConfig{
				Branch:              branchName,
				CommitMessageFormat: "{{.Action}} {{.Dotfile}}",
			},
		},
		Dotfiles: make(map[string]string),
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("error creating config directory: %w", err)
	}

	if err := SaveConfig(configPath, config); err != nil {
		return nil, fmt.Errorf("error saving default config: %w", err)
	}

	return config, nil
}
