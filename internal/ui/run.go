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
	storePath := filepath.Join(app.StoragePath, app.Dotfile.Name)
	// if dir not exist, Create the necessary directories in storePath if they don't exist
	_, err := os.Stat(storePath)
	if os.IsNotExist(err) {
		err := system.CopyDirectory(app.Dotfile.Path, storePath)
		if err != nil {
			fmt.Printf("Error creating directories in storePath: %v\n", err)
			return
		}
	}
	changed, err := system.DiffFiles(app.Dotfile.Path, storePath)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return
	}
	if changed {
		fmt.Println(changedStyle.Render(fmt.Sprintf("Changes detected in '%s'. Saving the updated dotfiles.", app.Dotfile.Name)))
		// Logic to save the modified files
	} else {
		fmt.Println(unchangedStyle.Render(fmt.Sprintf("No changes detected in '%s' since the last update.", app.Dotfile.Name)))
	}
}

func RunListView(app *app.App) {
	// We need to list the dotfiles before we can display them
	dotfiles := app.GetDotfiles()
	if len(dotfiles) == 0 {
		fmt.Println("No dotfiles found. Please add some dotfiles first.")
		return
	}

	model := InitListModel(*app.Config, app)
	if finalModel, err := tea.NewProgram(model).Run(); err == nil {
		finalListModel, ok := finalModel.(ListModel)
		if ok {
			fmt.Println(finalListModel.Table.View())
		} else {
			fmt.Println("Erreur lors de l'exécution du programme :", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Erreur lors de l'exécution du programme :", err)
		os.Exit(1)
	}
}
