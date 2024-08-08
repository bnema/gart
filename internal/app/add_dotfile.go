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
	app.mu.Lock()
	defer app.mu.Unlock()

	path = app.expandHomeDir(path)

	if strings.HasSuffix(path, "*") {
		app.addMultipleDotfiles(path, name)
	} else {
		app.addSingleDotfile(path, name)
	}
}

// expandHomeDir replaces the ~ in a path with the user's home directory
func (app *App) expandHomeDir(path string) string {
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

// addMultipleDotfiles adds multiple dotfiles to the storage folder
func (app *App) addMultipleDotfiles(path, name string) {
	directoryPath := strings.TrimSuffix(path, "*")
	dirs, err := filepath.Glob(filepath.Join(directoryPath, ".*"))
	if err != nil {
		fmt.Printf("Error using filepath.Glob: %v\n", err)
		return
	}

	dirChan := make(chan string, len(dirs))
	app.populateDirChannel(dirs, dirChan)

	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go app.workerCopyFiles(dirChan, &wg, directoryPath, name)
	}
	wg.Wait()

	app.updateConfig(name, path)
}

// populateDirChannel populates the directory channel with directories
func (app *App) populateDirChannel(dirs []string, dirChan chan<- string) {
	for _, dir := range dirs {
		if app.shouldIgnoreDir(dir) {
			fmt.Printf("Ignored directory: %s\n", dir)
			continue
		}
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirChan <- dir
		}
	}
	close(dirChan)
}

// shouldIgnoreDir checks if a directory should be ignored
func (app *App) shouldIgnoreDir(dir string) bool {
	return (filepath.Base(dir) == ".config" && strings.Contains(dir, "gart")) || filepath.Base(dir) == ".local"
}

// workerCopyFiles is a worker function that copies files in a directory
func (app *App) workerCopyFiles(dirChan <-chan string, wg *sync.WaitGroup, directoryPath, name string) {
	defer wg.Done()
	for dirPath := range dirChan {
		app.copyFilesInDir(dirPath, directoryPath, name)
	}
}

// copyFilesInDir copies all files in a directory to the storage folder
func (app *App) copyFilesInDir(dirPath, directoryPath, name string) {
	err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Size() > 1024*1024 {
			fmt.Printf("Ignored file due to size limit: %s\n", filePath)
			return nil
		}
		return app.copyFile(filePath, directoryPath, name, info)
	})
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}

// copyFile copies a file to the storage folder
func (app *App) copyFile(filePath, directoryPath, name string, info os.FileInfo) error {
	relPath, _ := filepath.Rel(directoryPath, filePath)
	destPath := filepath.Join(app.StorePath, name, relPath)
	if info.IsDir() {
		return os.MkdirAll(destPath, info.Mode())
	}
	return system.CopyFile(filePath, destPath)
}

// addSingleDotfile adds a single dotfile to the store
func (app *App) addSingleDotfile(path, name string) {
	cleanedPath := filepath.Clean(path)
	fmt.Printf("Adding dotfile: %s\n", cleanedPath)

	storePath := filepath.Join(app.StorePath, name)
	fmt.Printf("Store path: %s\n", storePath)

	err := system.CopyDirectory(cleanedPath, storePath)
	if err != nil {
		fmt.Printf("Error copying directory: %v\n", err)
		return
	}

	app.updateConfig(name, cleanedPath)
}

// updateConfig updates the config file with the new dotfile
func (app *App) updateConfig(name, path string) {
	err := config.AddDotfileToConfig(app.ConfigFilePath, name, path)
	if err != nil {
		fmt.Printf("Error adding dotfile to config: %v\n", err)
		return
	}

	if app.Config.Dotfiles == nil {
		app.Config.Dotfiles = make(map[string]string)
	}
	app.Config.Dotfiles[name] = path

	fmt.Printf("Dotfile added: %s\n", name)
}
