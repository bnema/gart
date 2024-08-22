package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"
)

func (app *App) RemoveDotFile(path string, name string) error {
	configFilePath := app.GetConfigFilePath()
	storagePath := app.StoragePath

	tree, err := toml.LoadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	dotfilesTree, ok := tree.Get("dotfiles").(*toml.Tree)
	if !ok {
		return fmt.Errorf("dotfiles section not found in config")
	}

	// The name in the config doesn't include the file extension
	configName := strings.TrimSuffix(name, filepath.Ext(name))

	if !dotfilesTree.Has(configName) {
		return fmt.Errorf("dotfile '%s' not found in config", configName)
	}

	dotfilesTree.Delete(configName)

	f, err := os.OpenFile(configFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(tree); err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	// Use the full path to determine the correct name in the storage directory
	storageItemName := filepath.Base(path)
	dotfileStoragePath := filepath.Join(storagePath, storageItemName)


	if err := os.RemoveAll(dotfileStoragePath); err != nil {
		return fmt.Errorf("error removing dotfile from storage: %w", err)
	}

	return nil
}
