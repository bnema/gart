package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Try nano if no editor is set
	if _, err := exec.LookPath("nano"); err == nil {
		return "nano"
	}
	return "vi" // Fallback to vi if no editor is set
}

func getConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		configHome = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configHome, "gart", "config.toml")
}

func getEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit Gart config file",
		Run: func(cmd *cobra.Command, args []string) {
			editor := getEditor()
			configPath := getConfigPath()

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
