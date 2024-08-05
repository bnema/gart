package app

import (
	"fmt"
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
}

// RunAddForm is the function that runs the add (new) dotfile form
func (app *App) RunAddForm() {
	model := ui.InitAddModel()
	if finalModel, err := tea.NewProgram(model).Run(); err == nil {
		finalAddModel, ok := finalModel.(ui.AddModel)
		if ok && finalAddModel.Inputs[0].Value() != "" && finalAddModel.Inputs[1].Value() != "" {
			app.AddDotfile(finalAddModel.Inputs[0].Value(), finalAddModel.Inputs[1].Value())
		} else {
			fmt.Println("Error: invalid inputs")
		}
	} else {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
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
	// We need to retreive the dotfiles list from the config file
	dotfiles := app.GetDotfilesList()

	model := ui.InitListModel(dotfiles)
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

// GetDotfilesList returns the dotfiles list from the config.toml file (after line [[dotfiles]])
func (app *App) GetDotfilesList() map[string]string {
	dotfiles, err := config.LoadConfig(app.ConfigFilePath)
	if err != nil {
		fmt.Printf("Erreur lors du chargement du fichier de configuration : %v\n", err)
		os.Exit(1)
	}

	return dotfiles

}
