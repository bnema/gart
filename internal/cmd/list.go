package cmd

import (
	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func getListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all dotfiles",
		Run: func(cmd *cobra.Command, args []string) {
			ui.RunListView(appInstance)
		},
	}
}
