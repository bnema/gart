package app

import (
	"fmt"
	"sync"

	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/git"
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

	config, err := config.LoadConfig(app.ConfigFilePath)
	if err != nil {
		app.ConfigError = err
		return err
	}

	app.Config = config
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

// GitInit initializes a Git repository in the storage path if enable = true
func (app *App) GitInit() error {
	if !app.Config.Settings.GitVersioning {
		return nil
	}

	return git.Init(app.StoragePath, app.Config.Settings.Git.Branch)
}

// GitCommitChanges commits changes to the Git repository
func (app *App) GitCommitChanges(action, dotfileName string) error {
	if !app.Config.Settings.GitVersioning {
		return nil // Git versioning is disabled, so we don't commit
	}
	err := git.CommitChanges(app.StoragePath, app.Config.Settings.Git.CommitMessageFormat, dotfileName, action)
	if err != nil {
		return err
	}

	if app.Config.Settings.Git.AutoPush {
		err = git.PushChanges(app.StoragePath)
		if err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}
