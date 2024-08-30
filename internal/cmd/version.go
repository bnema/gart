package cmd

import (
	"fmt"

	"github.com/bnema/gart/internal/version"
	"github.com/spf13/cobra"
)

func getVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Gart",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Full())
		},
	}
}
