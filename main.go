package main

import (
	"fmt"
	"os"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/cmd"
	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/git"
	"github.com/bnema/gart/internal/system"
)

func main() {
	_, configPath, err := system.GetConfigPaths()
	if err != nil {
		fmt.Println("Error getting config paths:", err)
		return
	}

	cfg, isFirstLaunch, err := checkFirstLaunch(configPath)
	if err != nil {
		fmt.Printf("Error during configuration check: %v\n", err)
		return
	}

	if isFirstLaunch {
		// do something
	}

	app := &app.App{
		StoragePath:    cfg.Settings.StoragePath,
		ConfigFilePath: configPath,
		Config:         cfg,
	}

	// Ensure the storage path exists
	if err := os.MkdirAll(app.StoragePath, 0755); err != nil {
		fmt.Printf("Error creating storage directory: %v\n", err)
		return
	}

	// If Git versioning is enabled, check if the repo exists and initialize if necessary
	if app.Config.Settings.GitVersioning {
		gitRepoExists, err := git.RepoExists(app.StoragePath)
		if err != nil {
			fmt.Printf("Error checking Git repository: %v\n", err)
			return
		}

		if !gitRepoExists {
			if err := initializeGitRepo(app); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	// Load the config
	err = app.LoadConfig()
	if err != nil {
		fmt.Println("Error loading the config:", err)
		return
	}

	// Begin command execution
	cmd.Execute(app)
}

// initializeGitRepo initializes a new Git repository and creates an initial commit
func initializeGitRepo(app *app.App) error {
	if err := git.Init(app.StoragePath, app.Config.Settings.Git.Branch); err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w", err)
	}
	fmt.Printf("Git repository initialized at %s (branch: %s)\n", app.StoragePath, app.Config.Settings.Git.Branch)

	if err := git.CommitChanges(app.StoragePath, "Initial commit", "", ""); err != nil {
		return fmt.Errorf("failed to perform initial commit: %w", err)
	}
	fmt.Println("Initial commit created successfully")
	return nil
}

// checkFirstLaunch checks if this is the first launch and prompts the user for Git versioning
func checkFirstLaunch(configPath string) (*config.Config, bool, error) {
	// Try to load the existing config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// If the error is because the file doesn't exist, create a default config
		if os.IsNotExist(err) {
			// Create a default config
			cfg, err = config.CreateDefaultConfig()
			if err != nil {
				return nil, false, fmt.Errorf("error creating default config: %w", err)
			}

			// Prompt for Git versioning
			enableGit, err := system.PromptForGitVersioning()
			if err != nil {
				return nil, false, fmt.Errorf("error prompting for Git versioning: %w", err)
			}

			cfg.Settings.GitVersioning = enableGit

			// Save the config with the user's Git versioning preference
			if err := config.SaveConfig(configPath, cfg); err != nil {
				return nil, false, fmt.Errorf("error saving initial config: %w", err)
			}

			return cfg, true, nil
		}
		// If it's any other error, return it
		return nil, false, fmt.Errorf("error loading config: %w", err)
	}

	// Config was loaded successfully, not first launch
	return cfg, false, nil
}
