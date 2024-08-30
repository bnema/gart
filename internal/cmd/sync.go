package cmd

import (
	"fmt"

	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func getSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [name]",
		Short: "Sync a dotfile or all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				syncAllDotfiles()
			} else {
				syncSingleDotfile(args[0])
			}
		},
	}
}

func syncAllDotfiles() {
	for name, path := range appInstance.Config.Dotfiles {
		appInstance.Dotfile.Name = name
		appInstance.Dotfile.Path = path
		ui.RunSyncView(appInstance)
	}
}

func syncSingleDotfile(name string) {
	path, ok := appInstance.Config.Dotfiles[name]
	if !ok {
		fmt.Printf("Dotfile '%s' not found.\n", name)
		return
	}
	appInstance.Dotfile.Name = name
	appInstance.Dotfile.Path = path
	ui.RunSyncView(appInstance)
}
