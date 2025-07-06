package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/gart/internal/security"
	"github.com/bnema/gart/internal/system"
	"github.com/pelletier/go-toml"
)

// Config represents the structure of the entire configuration file
type Config struct {
	Settings        SettingsConfig      `toml:"settings"`
	Dotfiles        map[string]string   `toml:"dotfiles"`
	DotfilesIgnores map[string][]string `toml:"dotfiles.ignores,omitempty"`
}

// SettingsConfig represents the general settings of the application
type SettingsConfig struct {
	StoragePath     string                   `toml:"storage_path"`
	GitVersioning   bool                     `toml:"git_versioning"`
	ReverseSyncMode bool                     `toml:"reverse_sync"`
	Git             GitConfig                `toml:"git"`
	Security        *security.SecurityConfig `toml:"security,omitempty"`
}

// GitConfig represents the structure of the git configuration
type GitConfig struct {
	Branch              string `toml:"branch"`
	CommitMessageFormat string `toml:"commit_message_format"`
	AutoPush            bool   `toml:"auto_push"`
}

// LoadConfig loads the configuration from the file
func LoadConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, err
	}

	var config Config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Ensure Dotfiles and Ignores are initialized
	if config.Dotfiles == nil {
		config.Dotfiles = make(map[string]string)
	}
	if config.DotfilesIgnores == nil {
		config.DotfilesIgnores = make(map[string][]string)
	}

	// Ensure Security config is initialized with defaults if not present
	if config.Settings.Security == nil {
		config.Settings.Security = security.DefaultSecurityConfig()
	}

	return &config, nil
}

// AddDotfileToConfig adds a new dotfile to the config file
func AddDotfileToConfig(configPath string, name, path string, ignores []string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error loading config file: %w", err)
		}
		config = &Config{
			Dotfiles: make(map[string]string),
			// Don't initialize DotfilesIgnores here unless ignores are provided
		}
	}

	config.Dotfiles[name] = path
	if len(ignores) > 0 {
		// Initialize DotfilesIgnores only when needed
		if config.DotfilesIgnores == nil {
			config.DotfilesIgnores = make(map[string][]string)
		}
		config.DotfilesIgnores[name] = ignores
	}

	return SaveConfig(configPath, config)
}

// SaveConfig saves the configuration to the file
func SaveConfig(configPath string, config *Config) error {
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0664)
}

// CreateDefaultConfig creates a default configuration
func CreateDefaultConfig() (*Config, error) {
	_, configPath, err := system.GetConfigPaths()
	if err != nil {
		return nil, fmt.Errorf("error getting config paths: %w", err)
	}

	gartDataDir, err := system.GetDataPaths()
	if err != nil {
		return nil, fmt.Errorf("error getting data paths: %w", err)
	}

	branchName, err := system.GetHostname()
	if err != nil {
		branchName = "main"
	}

	config := &Config{
		Settings: SettingsConfig{
			StoragePath:     filepath.Join(gartDataDir, "store"),
			ReverseSyncMode: false,
			GitVersioning:   false,
			Git: GitConfig{
				Branch:              branchName,
				CommitMessageFormat: "{{.Action}} {{.Dotfile}}",
				AutoPush:            false,
			},
			Security: security.DefaultSecurityConfig(),
		},
		Dotfiles: make(map[string]string),
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return nil, fmt.Errorf("error creating config directory: %w", err)
	}

	if err := SaveConfig(configPath, config); err != nil {
		return nil, fmt.Errorf("error saving default config: %w", err)
	}

	fmt.Printf("Created default configuration at %s\n", configPath)

	return config, nil
}

// UpdateDotfileIgnores updates the ignores for an existing dotfile in the config file
func UpdateDotfileIgnores(configPath string, name string, ignores []string) error {
	tree, err := toml.LoadFile(configPath)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	ignoresTree, ok := tree.Get("dotfiles.ignores").(*toml.Tree)
	if !ok {
		ignoresTree, _ = toml.Load("")
		tree.Set("dotfiles.ignores", ignoresTree)
	}
	ignoresTree.Set(name, ignores)

	f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	encoder := toml.NewEncoder(f)
	err = encoder.Encode(tree)
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	return nil
}
