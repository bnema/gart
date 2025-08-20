package app

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

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
	gitRepo        git.GitRepository
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

// SetGitRepository sets the git repository implementation to use
func (app *App) SetGitRepository(repo git.GitRepository) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.gitRepo = repo
}

// getOrCreateGitRepository returns the git repository, creating it if needed
func (app *App) getOrCreateGitRepository() (git.GitRepository, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	
	if app.gitRepo == nil {
		repo, err := git.NewRepository(app.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create git repository: %w", err)
		}
		app.gitRepo = repo
	}
	
	return app.gitRepo, nil
}

// GitInit initializes a Git repository in the storage path if enable = true
func (app *App) GitInit() error {
	if !app.Config.Settings.GitVersioning {
		return nil
	}

	repo, err := app.getOrCreateGitRepository()
	if err != nil {
		return err
	}

	return repo.Init(app.Config.Settings.Git.Branch)
}

// GitCommitChanges commits changes to the Git repository
func (app *App) GitCommitChanges(action, dotfileName string) error {
	if !app.Config.Settings.GitVersioning {
		return nil // Git versioning is disabled, so we don't commit
	}

	repo, err := app.getOrCreateGitRepository()
	if err != nil {
		return err
	}

	// Add all changes
	err = repo.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Parse the commit message template
	tmpl, err := template.New("commit").Parse(app.Config.Settings.Git.CommitMessageFormat)
	if err != nil {
		return fmt.Errorf("failed to parse commit message template: %w", err)
	}

	// Create a buffer to store the executed template
	var buf bytes.Buffer

	// Execute the template with the dotfile name and action
	err = tmpl.Execute(&buf, struct {
		Dotfile string
		Action  string
	}{
		Dotfile: dotfileName,
		Action:  action,
	})
	if err != nil {
		return fmt.Errorf("failed to execute commit message template: %w", err)
	}

	// Get the formatted commit message
	commitMessage := buf.String()

	// Commit the changes
	err = repo.Commit(commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push if auto_push is enabled
	if app.Config.Settings.Git.AutoPush {
		err = repo.Push()
		if err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}

func (app *App) UpdateDotfileIgnores(name string, ignores []string) error {
	if _, ok := app.Config.Dotfiles[name]; !ok {
		return fmt.Errorf("dotfile '%s' not found", name)
	}

	app.Config.DotfilesIgnores[name] = ignores
	return config.UpdateDotfileIgnores(app.ConfigFilePath, name, ignores)
}
