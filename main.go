package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/system"
	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func main() {
	configPath, err := system.GetOSConfigPath()
	if err != nil {
		fmt.Println("Error getting the config path:", err)
		return
	}

	app := &app.App{
		ConfigFilePath: filepath.Join(configPath, "config.toml"),
		StoragePath:    filepath.Join(configPath, ".store"),
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
		Short: "Update a dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			updateDotfile := func(name, path string) {
				app.Dotfile.Name = name
				app.Dotfile.Path = path
				ui.RunUpdateView(app)
			}

			if len(args) == 0 {
				// Update all dotfiles
				for name, path := range app.Config.Dotfiles {
					updateDotfile(name, path)
				}
			} else {
				// Update a specific dotfile
				name := args[0]
				path, ok := app.Config.Dotfiles[name]
				if !ok {
					fmt.Printf("Dotfile '%s' not found.\n", name)
					return
				}
				updateDotfile(name, path)
			}
		},
	}

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
