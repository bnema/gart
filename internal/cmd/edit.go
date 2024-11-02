package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bnema/gart/internal/system"
	"github.com/spf13/cobra"
)

func getEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit Gart config file",
		Run: func(cmd *cobra.Command, args []string) {
			editor := system.GetEditor()
			_, configPath, err := system.GetConfigPaths()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting config paths: %v\n", err)
				os.Exit(1)
			}

			// Execute the editor
			editorCmd := exec.Command(editor, configPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
				os.Exit(1)
			}
		},
	}
}
