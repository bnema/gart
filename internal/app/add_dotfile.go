package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/system"
)

func (app *App) AddDotfile(path string, dotfileName string) error {
	path = expandHomeDir(path)

	// Update the dotfile path to the expanded path
	app.Dotfile.Name = dotfileName
	app.Dotfile.Path = path

	// Check if the path is a directory
	if isDir(path) {
		app.addDotfileDir()
	} else {
		app.addDotfileFile()
	}

	// Commit changes
	if err := app.GitCommitChanges("Add", dotfileName); err != nil {
		return fmt.Errorf("error committing changes for %s: %w", dotfileName, err)
	}

	return nil
}

// expandHomeDir replaces the ~ in a path with the user's home directory
func expandHomeDir(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			return path
		}
		return home + path[1:]
	}
	return path
}

// addDotfileDir adds the dotfile directory to the storage
func (app *App) addDotfileDir() {
	path := app.Dotfile.Path
	name := app.Dotfile.Name
	cleanedPath := filepath.Clean(path)
	fmt.Printf("Adding dotfiles: %s\n", cleanedPath)

	storePath := filepath.Join(app.StoragePath, name)
	fmt.Printf("Storage path: %s\n", storePath)

	err := system.CopyDirectory(cleanedPath, storePath)
	if err != nil {
		fmt.Printf("Error copying directory: %v\n", err)
		return
	}

	err = app.updateConfig(cleanedPath)
	if err != nil {
		fmt.Printf("Error updating config: %v\n", err)
		return
	}
}

// addDotfileFile adds the dotfile file to the storage
func (app *App) addDotfileFile() {
	path := app.Dotfile.Path
	cleanedPath := filepath.Clean(path)
	fmt.Printf("Adding dotfile: %s\n", cleanedPath)

	// Use filepath.Base to get the filename with extension
	fileName := filepath.Base(cleanedPath)
	storePath := filepath.Join(app.StoragePath, fileName)

	fmt.Printf("Storage path: %s\n", storePath)

	// Ensure the store directory exists
	if err := os.MkdirAll(app.StoragePath, os.ModePerm); err != nil {
		fmt.Printf("Error creating store directory: %v\n", err)
		return
	}

	err := system.CopyFile(cleanedPath, storePath)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		return
	}
	err = app.updateConfig(cleanedPath)
	if err != nil {
		fmt.Printf("Error updating config: %v\n", err)
		return
	}
}

// updateConfig updates the config file with the new dotfile
func (app *App) updateConfig(cleanedPath string) error {

	app.Config.Dotfiles[app.Dotfile.Name] = cleanedPath

	err := config.AddDotfileToConfig(app.ConfigFilePath, app.Dotfile.Name, cleanedPath)
	if err != nil {
		return fmt.Errorf("Error adding dotfile to config: %v\n", err)
	}

	if app.Config.Dotfiles == nil {
		app.Config.Dotfiles = make(map[string]string)
	}

	fmt.Printf("%s added successfully!\n", app.Dotfile.Name)
	return nil
}

// isDir checks if a path is a directory and return a boolean
func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
