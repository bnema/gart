package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/system"
	tea "github.com/charmbracelet/bubbletea"
)

// RunUpdateView is the function that runs the update (edit) dotfile view
func RunUpdateView(app *app.App) {
	sourcePath := app.Dotfile.Path

	// Check if the source is a file or directory
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		fmt.Printf("Error accessing source path: %v\n", err)
		return
	}

	var storePath string
	if sourceInfo.IsDir() {
		// For directories, use the dotfile name as is
		storePath = filepath.Join(app.StoragePath, app.Dotfile.Name)
	} else {
		// For files, include the file extension in the store path
		ext := filepath.Ext(sourcePath)
		storePath = filepath.Join(app.StoragePath, app.Dotfile.Name+ext)
	}

	// Check for changes before copying
	changed, err := system.DiffFiles(sourcePath, storePath)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return
	}

	if changed {
		fmt.Print(changedStyle.Render(fmt.Sprintf("Changes detected in '%s'. Updating...", app.Dotfile.Name)))

		var err error
		if sourceInfo.IsDir() {
			// Handle directory
			err = system.CopyDirectory(sourcePath, storePath)
		} else {
			// Handle single file
			err = os.MkdirAll(filepath.Dir(storePath), 0755)
			if err == nil {
				err = system.CopyFile(sourcePath, storePath)
			}
		}

		if err != nil {
			fmt.Printf(" %s\n", errorStyle.Render(fmt.Sprintf("Error: %v", err)))
			return
		}

		// Commit changes
		if err := app.GitCommitChanges("Update", app.Dotfile.Name); err != nil {
			fmt.Printf(" %s\n", errorStyle.Render(fmt.Sprintf("Error committing changes: %v", err)))
			return
		}

		fmt.Printf(" %s\n", successStyle.Render("Success!"))
	} else {
		fmt.Println(unchangedStyle.Render(fmt.Sprintf("No changes detected in '%s' since the last update.", app.Dotfile.Name)))
	}
}

func RunListView(app *app.App) {
	dotfiles := app.GetDotfiles()
	if len(dotfiles) == 0 {
		fmt.Println("No dotfiles found. Please add some dotfiles first.")
		return
	}

	model := InitListModel(*app.Config, app)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
