package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bnema/Gart/internal/config"
	"github.com/bnema/Gart/internal/system"
	"github.com/bnema/Gart/internal/ui"
	"github.com/bnema/Gart/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type App struct {
	ListModel      *ui.ListModel
	AddModel       *ui.AddModel
	ConfigFilePath string
	StorePath      string
	Config         *config.Config
	ConfigError    error
}

// RunUpdateView is the function that runs the update (edit) dotfile view
func (app *App) RunUpdateView(name, path string) {
	storePath := filepath.Join(app.StorePath, name)

	// if dir not exist, Create the necessary directories in storePath if they don't exist
	_, err := os.Stat(storePath)
	if os.IsNotExist(err) {

		err := system.CopyDirectory(path, storePath)
		if err != nil {
			fmt.Printf("Error creating directories in storePath: %v\n", err)
			return

		}
	}

	changed, err := utils.DiffFiles(path, storePath)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return
	}

	if changed {
		fmt.Printf("Changes detected in '%s'. Saving the updated dotfiles.\n", name)
		// Logic to save the modified files
	} else {
		fmt.Printf("No changes detected in '%s' since the last update.\n", name)
	}
}

func (app *App) RunListView() {
	// We need to load the config before we can run the list view
	config, err := config.LoadConfig(app.ConfigFilePath)
	if err != nil {
		log.Fatal(err)
	}

	app.Config = &config

	model := ui.InitListModel(*app.Config)
	if finalModel, err := tea.NewProgram(model).Run(); err == nil {
		finalListModel, ok := finalModel.(ui.ListModel)
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
