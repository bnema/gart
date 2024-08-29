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

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	app := &app.App{
		ConfigFilePath: cfg.Settings.ConfigFilePath,
		StoragePath:    cfg.Settings.StoragePath,
		Config:         cfg,
	}

	// Ensure the storage path exists
	if err := os.MkdirAll(app.StoragePath, 0755); err != nil {
		fmt.Printf("Error creating storage directory: %v\n", err)
		return
	}

	gitInitialized := false
	if !cfg.Settings.GitVersioning {
		enableGit, err := system.PromptForGitVersioning()
		if err != nil {
			fmt.Println("Error prompting for Git versioning:", err)
			return
		}

		if enableGit {
			cfg.Settings.GitVersioning = true
			if err := git.Init(app.StoragePath, cfg.Settings.Git.Branch); err != nil {
				fmt.Printf("Error initializing Git repository: %v\n", err)
				return
			}
			gitInitialized = true
			fmt.Println("Git repository initialized successfully.")
		}

		if err := config.SaveConfig(configPath, cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}
	}

	// Load the config
	err = app.LoadConfig()
	if err != nil {
		fmt.Println("Error loading the config:", err)
		return
	}

	// If Git was just initialized, perform an initial commit
	if gitInitialized {
		if err := git.CommitChanges(app.StoragePath, "Initial commit", "", ""); err != nil {
			fmt.Printf("Error performing initial commit: %v\n", err)
		} else {
			fmt.Println("Initial commit performed successfully.")
		}
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
