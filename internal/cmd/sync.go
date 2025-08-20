package cmd

import (
	"fmt"

	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func getSyncCmd() *cobra.Command {
	var skipSecurity bool
	
	cmd := &cobra.Command{
		Use:   "sync [name]",
		Short: "Sync a dotfile or all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				syncAllDotfiles(skipSecurity)
			} else {
				syncSingleDotfile(args[0], skipSecurity)
			}
		},
	}
	
	cmd.Flags().BoolVar(&skipSecurity, "no-security", false, "Skip security scanning")
	
	return cmd
}

func syncAllDotfiles(skipSecurity bool) {
	for name, path := range appInstance.Config.Dotfiles {
		appInstance.Dotfile.Name = name
		appInstance.Dotfile.Path = path
		// Get ignores for this dotfile
		ignores := appInstance.Config.DotfilesIgnores[name]
		if !ui.RunSyncView(appInstance, ignores, skipSecurity) {
			// User aborted sync, stop processing remaining dotfiles
			break
		}
	}
}

func syncSingleDotfile(name string, skipSecurity bool) {
	path, ok := appInstance.Config.Dotfiles[name]
	if !ok {
		fmt.Printf("Dotfile '%s' not found.\n", name)
		return
	}
	appInstance.Dotfile.Name = name
	appInstance.Dotfile.Path = path
	// Get ignores for this dotfile
	ignores := appInstance.Config.DotfilesIgnores[name]
	ui.RunSyncView(appInstance, ignores, skipSecurity)
}
