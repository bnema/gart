package app

import (
	"fmt"
	"os"

	"github.com/bnema/gart/internal/system"
	"github.com/pelletier/go-toml"
)

func (app *App) RemoveDotFile(path string, name string) error {

	// Get the configuration file path
	configFilePath := app.GetConfigFilePath()
	storagePath := app.StoragePath

	// Load the existing configuration
	tree, err := toml.LoadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	// Get the dotfiles tree
	dotfilesTree, ok := tree.Get("dotfiles").(*toml.Tree)
	if !ok {
		return fmt.Errorf("dotfiles section not found in config")
	}

	// Check if the dotfile exists
	if !dotfilesTree.Has(name) {
		return fmt.Errorf("dotfile '%s' not found in config", app.Dotfile.Name)
	}

	// Remove the dotfile entry
	dotfilesTree.Delete(name)

	// Open the file for writing
	f, err := os.OpenFile(configFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	// Encode and write the updated configuration
	encoder := toml.NewEncoder(f)
	err = encoder.Encode(tree)
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	err = system.RemoveDirectory(storagePath)
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	return nil
}
