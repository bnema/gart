package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"
)

// isDirectory checks if the given path is a directory
func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// findKeyAndPath finds the key and path in the dotfiles tree
func findKeyAndPath(dotfilesTree *toml.Tree, path, name string) (string, string) {
	for _, key := range dotfilesTree.Keys() {
		value, _ := dotfilesTree.Get(key).(string)
		if value == path || strings.EqualFold(key, name) {
			return key, value
		}
	}
	return "", ""
}

// getStorageItemName determines the name of the item in storage
func getStorageItemName(path, key string) (string, error) {
	isDir, err := isDirectory(path)
	if err != nil {
		return "", fmt.Errorf("error checking if path is directory: %w", err)
	}

	if isDir {
		return key, nil
	}
	return filepath.Base(path), nil
}

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

	keyToRemove, pathToRemove := findKeyAndPath(dotfilesTree, path, name)

	if keyToRemove == "" {
		return fmt.Errorf("dotfile with path '%s' or name '%s' not found in config", path, name)
	}

	dotfilesTree.Delete(keyToRemove)

	f, err := os.OpenFile(configFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(tree); err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	storageItemName, err := getStorageItemName(pathToRemove, keyToRemove)
	if err != nil {
		return err
	}

	dotfileStoragePath := filepath.Join(storagePath, storageItemName)

	if err := os.RemoveAll(dotfileStoragePath); err != nil {
		return fmt.Errorf("error removing dotfile from storage: %w", err)
	}

	return nil
}
