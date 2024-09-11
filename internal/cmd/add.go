package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/ui"
	"github.com/spf13/cobra"
)

func getAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [path] [name]",
		Short: "Add a new dotfile or folder",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Invalid arguments. Usage: add [path] opt:[name]")
				return
			}

			path := args[0]
			var name string

			if len(args) == 1 {
				name = getNameFromPath(path)
			} else if len(args) == 2 {
				name = args[1]
			} else {
				fmt.Println("Invalid arguments. Usage: add [path] opt:[name]")
				return
			}

			ui.RunAddDotfileView(appInstance, path, name)
		},
	}
}

func getNameFromPath(path string) string {
	name := filepath.Base(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return name
	}

	if !fileInfo.IsDir() {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}

	return name
}
