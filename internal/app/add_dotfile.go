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
	path = app.ExpandHomeDir(path)

	app.Dotfile.Name = dotfileName
	app.Dotfile.Path = path

	if app.IsDir(path) {
		return app.addDotfileDir()
	}
	return app.addDotfileFile()
}

func (app *App) ExpandHomeDir(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func (app *App) addDotfileDir() error {
	cleanedPath := filepath.Clean(app.Dotfile.Path)
	storePath := filepath.Join(app.StoragePath, app.Dotfile.Name)

	if err := system.CopyDirectory(cleanedPath, storePath); err != nil {
		return fmt.Errorf("error copying directory: %w", err)
	}

	return app.UpdateConfig(app.Dotfile.Name, cleanedPath)
}

func (app *App) addDotfileFile() error {
	cleanedPath := filepath.Clean(app.Dotfile.Path)
	fileName := filepath.Base(cleanedPath)
	storePath := filepath.Join(app.StoragePath, fileName)

	if err := os.MkdirAll(app.StoragePath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating store directory: %w", err)
	}

	if err := system.CopyFile(cleanedPath, storePath); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return app.UpdateConfig(app.Dotfile.Name, cleanedPath)
}

func (app *App) UpdateConfig(dotfileName, cleanedPath string) error {
	app.Config.Dotfiles[dotfileName] = cleanedPath

	if err := config.AddDotfileToConfig(app.ConfigFilePath, dotfileName, cleanedPath); err != nil {
		return fmt.Errorf("error adding dotfile to config: %w", err)
	}

	if app.Config.Dotfiles == nil {
		app.Config.Dotfiles = make(map[string]string)
	}

	return nil
}

func (app *App) IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func (app *App) CopyDirectory(src, dst string) error {
	return system.CopyDirectory(src, dst)
}

func (app *App) CopyFile(src, dst string) error {
	return system.CopyFile(src, dst)
}
