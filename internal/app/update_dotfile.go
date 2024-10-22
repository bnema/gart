package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/gart/internal/system"
)

// UpdateDotfile updates a single dotfile
func (app *App) UpdateDotfile(name, path string) error {
	app.Dotfile.Name = name
	app.Dotfile.Path = path

	// Get the destination path in the storage
	destPath := filepath.Join(app.StoragePath, name)

	// Ensure the destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get the ignores for this dotfile
	ignores := app.Config.DotfilesIgnores[name]

	// Copy the file or directory
	if err := system.CopyPath(path, destPath, ignores); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", path, destPath, err)
	}

	return nil
}

// UpdateAllDotfiles updates all dotfiles in the configuration
func (app *App) UpdateAllDotfiles() error {
	for name, path := range app.Config.Dotfiles {
		if err := app.UpdateDotfile(name, path); err != nil {
			return fmt.Errorf("error updating dotfile %s: %w", name, err)
		}
	}
	return nil
}

// UpdateDotfiles updates either all dotfiles or a specific one based on the provided name
func (app *App) UpdateDotfiles(name string) error {
	if name == "" {
		return app.UpdateAllDotfiles()
	}

	path, ok := app.Config.Dotfiles[name]
	if !ok {
		return fmt.Errorf("dotfile '%s' not found", name)
	}
	return app.UpdateDotfile(name, path)
}
