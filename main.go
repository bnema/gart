package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/git"
	"github.com/bnema/gart/internal/system"
	"github.com/bnema/gart/internal/ui"
	"github.com/bnema/gart/internal/version"
	"github.com/spf13/cobra"
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
		fmt.Println("First launch detected.New configuration created.")
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

	rootCmd := &cobra.Command{
		Use:   "gart",
		Short: "Gart is a dotfile manager",
		Long:  `Gart is a command-line tool for managing dotfiles.`,
	}

	// Show at root level the list of dotfiles
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			ui.RunListView(app)
		},
	}

	addCmd := &cobra.Command{
		Use:   "add [path] [name]",
		Short: "Add a new dotfile folder",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// If no arguments are provided, show the usage
				fmt.Println("Invalid arguments. Usage: add [path] opt:[name]")
			} else if len(args) == 1 {
				// If only the path is provided, use the last part of the path as the name
				path := args[0]
				name := filepath.Base(path)

				// In case of file use the file name without extension
				// Check if the path is a file
				fileInfo, err := os.Stat(path)
				if err != nil {
					fmt.Printf("Error accessing path: %v\n", err)
					return
				}

				if !fileInfo.IsDir() {
					// If it's a file, use the file name without extension
					name = strings.TrimSuffix(name, filepath.Ext(name))
				}

				err = app.AddDotfile(path, name)
				if err != nil {
					fmt.Printf("Error adding dotfile: %v\n", err)
					return
				}

			} else if len(args) == 2 {
				// If both path and name are provided, use them as is
				path := args[0]
				name := args[1]

				err := app.AddDotfile(path, name)
				if err != nil {
					fmt.Printf("Error adding dotfile: %v\n", err)
					return
				}

			} else {
				fmt.Println("Invalid arguments. Usage: add [path] opt:[name]")
			}
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update a dotfile or all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Update all dotfiles
				for name, path := range app.Config.Dotfiles {
					app.Dotfile.Name = name
					app.Dotfile.Path = path
					ui.RunUpdateView(app)
				}
			} else {
				// Update a specific dotfile
				name := args[0]
				path, ok := app.Config.Dotfiles[name]
				if !ok {
					fmt.Printf("Dotfile '%s' not found.\n", name)
					return
				}
				app.Dotfile.Name = name
				app.Dotfile.Path = path
				ui.RunUpdateView(app)
			}

		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Gart",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Full())
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)

		// Handle the error
		if app.ConfigError != nil {
			fmt.Println("Error reading the config file:", app.ConfigError)
		}

		// Exit with an error code
		return

	}
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
		// If the error is because the file doesn't exist, it's the first launch
		if os.IsNotExist(err) {
			// Create a default config
			cfg, err = config.CreateDefaultConfig(configPath)
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
