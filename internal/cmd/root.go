package cmd

import (
	"fmt"
	"os"

	"github.com/bnema/gart/internal/app"
	"github.com/spf13/cobra"
)

var (
	rootCmd     *cobra.Command
	appInstance *app.App
)

func init() {
	rootCmd = &cobra.Command{
		Use:   "gart",
		Short: "Gart is a dotfile manager",
		Long:  `Gart is a command-line tool for managing dotfiles.`,
	}
}

func Execute(a *app.App) {
	appInstance = a

	rootCmd.AddCommand(getVersionCmd())
	rootCmd.AddCommand(getAddCmd())
	rootCmd.AddCommand(getSyncCmd())
	rootCmd.AddCommand(getListCmd())
	rootCmd.AddCommand(getEditCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		if a.ConfigError != nil {
			fmt.Println("Error reading the config file:", a.ConfigError)
		}
		os.Exit(1)
	}
}
