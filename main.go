package main

import (
	"fmt"
	"path/filepath"

	"github.com/bnema/Gart/internal/app"
	"github.com/bnema/Gart/internal/config"
	"github.com/bnema/Gart/internal/utils"
	"github.com/spf13/cobra"
)

func main() {
	configPath, err := utils.GetOSConfigPath()
	if err != nil {
		fmt.Println("Error getting the config path:", err)
		return
	}

	app := &app.App{
		ConfigFilePath: filepath.Join(configPath, "config.toml"),
		StorePath:      filepath.Join(configPath, ".store"),
	}

	// Try to load the configuration file
	config, configError := config.LoadConfig(app.ConfigFilePath)
	app.Config = &config
	app.ConfigError = configError

	// Print the dotfiles
	fmt.Printf("Dotfiles: %v\n", app.Config.Dotfiles)

	var rootCmd = &cobra.Command{
		Use:   "gart",
		Short: "Gart is a dotfile manager",
		Long:  `Gart is a command-line tool for managing dotfiles.`,
	}

	// Show at root level the list of dotfiles
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			app.RunListView()
		},
	}

	var addCmd = &cobra.Command{
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
				app.AddDotfile(path, name)
			} else if len(args) == 2 {
				// If both path and name are provided, use them as is
				path := args[0]
				name := args[1]
				app.AddDotfile(path, name)
			} else {
				fmt.Println("Invalid arguments. Usage: add [path] opt:[name]")
			}
		},
	}

	var updateCmd = &cobra.Command{
		Use:   "update [name]",
		Short: "Update a dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Update all dotfiles
				for name, path := range app.Config.Dotfiles {
					app.RunUpdateView(name, path)
				}
			} else {
				// Update a specific dotfile
				name := args[0]
				path, ok := app.Config.Dotfiles[name]
				if !ok {
					fmt.Printf("Dotfile '%s' not found.\n", name)
					return
				}
				app.RunUpdateView(name, path)
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
