package cmd

import (
	"fmt"

	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func getUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [name]",
		Short: "Update a dotfile or all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				updateAllDotfiles()
			} else {
				updateSingleDotfile(args[0])
			}
		},
	}
}

func updateAllDotfiles() {
	for name, path := range appInstance.Config.Dotfiles {
		appInstance.Dotfile.Name = name
		appInstance.Dotfile.Path = path
		ui.RunUpdateView(appInstance)
	}
}

func updateSingleDotfile(name string) {
	path, ok := appInstance.Config.Dotfiles[name]
	if !ok {
		fmt.Printf("Dotfile '%s' not found.\n", name)
		return
	}
	appInstance.Dotfile.Name = name
	appInstance.Dotfile.Path = path
	ui.RunUpdateView(appInstance)
}
