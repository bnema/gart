package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bnema/Gart/internal/config"
	"github.com/bnema/Gart/internal/system"
)

func (app *App) AddDotfile(path, name string) {

	// If the path starts with ~, replace it with the user's home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			return
		}
		path = home + path[1:]
	}

	if strings.HasSuffix(path, "*") {
		directoryPath := strings.TrimSuffix(path, "*")

		// Use filepath.Glob to get the list of directories matching the pattern
		dirs, err := filepath.Glob(filepath.Join(directoryPath, ".*"))
		if err != nil {
			fmt.Printf("Error using filepath.Glob: %v\n", err)
			return
		}

		// Create a wait group to wait for all worker goroutines to finish
		var wg sync.WaitGroup

		// Determine the number of worker goroutines based on available CPU cores
		numWorkers := runtime.NumCPU()

		// Create a buffered channel to hold the directory paths
		dirChan := make(chan string, len(dirs))

		// Send the directory paths to the channel
		for _, dir := range dirs {
			if (filepath.Base(dir) == ".config" && strings.Contains(dir, "gart")) || filepath.Base(dir) == ".local" {
				fmt.Printf("Ignored directory: %s\n", dir)
				continue
			}
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				dirChan <- dir
			}
		}
		close(dirChan)

		// Start the worker goroutines
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for dirPath := range dirChan {
					err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Check if the file size is within the limit (1024*1024 bytes)
						if !info.IsDir() && info.Size() > 1024*1024 {
							fmt.Printf("Ignored file due to size limit: %s\n", filePath)
							return nil
						}

						relPath, _ := filepath.Rel(directoryPath, filePath)
						destPath := filepath.Join(app.StorePath, name, relPath)

						if info.IsDir() {
							err := os.MkdirAll(destPath, info.Mode())
							if err != nil {
								return err
							}
						} else {
							err := system.CopyFile(filePath, destPath)
							if err != nil {
								return err
							}
						}

						return nil
					})

					if err != nil {
					}

					fmt.Printf("Dotfiles added: %s\n", name)

				}
			}()
		}

		// Wait for all worker goroutines to finish
		wg.Wait()

		// Start a new goroutine to save the configuration
		go func() {
			err := config.SaveConfig(app.ConfigFilePath, *app.Config)
			if err != nil {
				fmt.Printf("Error saving configuration: %v\n", err)
				return
			}
			fmt.Printf("Configuration saved successfully\n")
		}()

	} else {

		cleanedPath := filepath.Clean(path)
		fmt.Printf("Adding dotfile: %s\n", cleanedPath)

		storePath := filepath.Join(app.StorePath, name)
		fmt.Printf("Store path: %s\n", storePath)

		err := system.CopyDirectory(cleanedPath, storePath)
		if err != nil {
			fmt.Printf("Error copying directory: %v\n", err)
			return
		}

		// Add the dotfile to the map of dotfiles
		if app.Config.Dotfiles == nil {
			app.Config.Dotfiles = make(map[string]string)
		}
		app.Config.Dotfiles[name] = cleanedPath

		// Save the updated configuration
		err = config.SaveConfig(app.ConfigFilePath, *app.Config)
		if err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			return
		}

		fmt.Printf("Dotfile added: %s\n", name)
	}
}
