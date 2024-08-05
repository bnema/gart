package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/Gart/internal/app"
	"github.com/bnema/Gart/internal/utils"
	"github.com/spf13/cobra"
)

func main() {
	configPath := utils.GetOSConfigPath()

	app := &app.App{
		ConfigFilePath: filepath.Join(configPath, "config.toml"),
		StorePath:      filepath.Join(configPath, ".store"),
	}

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
		Short: "Add a new dotfile",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// If no arguments are provided, start the form
				app.RunAddForm()
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
				for name, path := range app.ListModel.Dotfiles {
					app.RunUpdateView(name, path)
				}
			} else {
				// Update a specific dotfile
				name := args[0]
				path, ok := app.ListModel.Dotfiles[name]
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
		os.Exit(1)
	}
}
