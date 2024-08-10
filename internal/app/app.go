package app

import (
	"sync"

	"github.com/bnema/gart/internal/config"
)

type App struct {
	ConfigFilePath string
	StoragePath    string
	Dotfile        Dotfile
	Config         *config.Config
	ConfigError    error
	mu             sync.RWMutex
}

type Dotfile struct {
	Name string
	Path string
}

func (app *App) LoadConfig() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	dotfiles, err := config.LoadDotfilesConfig(app.ConfigFilePath)
	if err != nil {
		app.ConfigError = err
		return err
	}

	app.Config = &config.Config{
		Dotfiles: dotfiles,
	}
	return nil
}

func (app *App) GetDotfiles() map[string]string {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.Config.Dotfiles
}

func (app *App) ReloadConfig() error {
	return app.LoadConfig()
}

func (app *App) GetConfigFilePath() string {
	return app.ConfigFilePath
}
