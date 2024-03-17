package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bnema/Gart/config"
	"github.com/bnema/Gart/system"
)

func (app *App) addDotfile(path, name string) {
	// Si le chemin commence par ~, remplacez-le par le répertoire personnel de l'utilisateur
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Erreur lors de l'obtention du répertoire personnel : %v\n", err)
			return
		}
		path = home + path[1:]
	}

	if strings.HasSuffix(path, "*") {
		directoryPath := strings.TrimSuffix(path, "*")

		// Créez un canal tamponné pour contenir les chemins des répertoires
		dirChan := make(chan string, 1000)

		// Créez un groupe d'attente pour attendre que toutes les goroutines des travailleurs se terminent
		var wg sync.WaitGroup

		// Déterminez le nombre de goroutines de travailleurs en fonction des cœurs CPU disponibles
		numWorkers := runtime.NumCPU()

		// Démarrez les goroutines des travailleurs
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for dirPath := range dirChan {
					dirName := filepath.Base(dirPath)
					if (dirName == ".config" && strings.Contains(dirPath, "gart")) || dirName == ".local" {
						fmt.Printf("Répertoire ignoré : %s\n", dirPath)
						continue
					}
					destPath := filepath.Join(app.StorePath, name, dirName)
					system.CopyDirectory(dirPath, destPath)
				}
			}()
		}

		// Parcourez l'arborescence des répertoires et envoyez les chemins des répertoires au canal
		err := filepath.Walk(directoryPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				dirChan <- filePath
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Erreur lors de la traversée du répertoire : %v\n", err)
			return
		}

		// Fermez le canal pour signaler aux travailleurs de s'arrêter
		close(dirChan)

		// Attendez que toutes les goroutines des travailleurs se terminent
		wg.Wait()
	} else {
		cleanedPath := filepath.Clean(path)

		storePath := filepath.Join(app.StorePath, name)
		err := system.CopyDirectory(cleanedPath, storePath)
		if err != nil {
			fmt.Printf("Erreur lors de la copie du répertoire : %v\n", err)
			return
		}

		app.ListModel.dotfiles[name] = cleanedPath
		err = config.SaveConfig(app.ConfigFilePath, app.ListModel.dotfiles)
		if err != nil {
			fmt.Printf("Erreur lors de l'enregistrement de la configuration : %v\n", err)
			return
		}

		fmt.Printf("Dotfile ajouté : %s\n", name)
	}
}
