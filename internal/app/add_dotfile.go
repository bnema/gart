package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/system"
)

func (app *App) AddDotfile(path string, name string) error {
	path = expandHomeDir(path)

	// Update the dotfile path to the expanded path
	app.Dotfile.Name = name
	app.Dotfile.Path = path

	app.addDotfileDir()

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
	fmt.Printf("Adding dotfile: %s\n", cleanedPath)

	storePath := filepath.Join(app.StoragePath, name)
	fmt.Printf("Store path: %s\n", storePath)

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

// updateConfig updates the config file with the new dotfile
func (app *App) updateConfig(cleanedPath string) error {

	// Add the dotfile to the array of dotfiles
	// TODO: rewrite this
	app.Config.Dotfiles[app.Dotfile.Name] = cleanedPath

	err := config.AddDotfileToConfig(app.ConfigFilePath, app.Dotfile.Name, cleanedPath)
	if err != nil {
		return fmt.Errorf("Error adding dotfile to config: %v\n", err)
	}

	if app.Config.Dotfiles == nil {
		app.Config.Dotfiles = make(map[string]string)
	}

	fmt.Printf("Dotfile added: %s\n", app.Dotfile.Name)
	return nil
}
