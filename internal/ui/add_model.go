package ui

import (
	"fmt"
	"path/filepath"

	"github.com/bnema/gart/internal/app"
)

func RunAddDotfileView(app *app.App, path string, dotfileName string) {
	path = app.ExpandHomeDir(path)
	cleanedPath := filepath.Clean(path)

	fmt.Printf("Adding dotfile %s... ", dotfileName)

	var err error
	if app.IsDir(path) {
		err = addDotfileDir(app, cleanedPath, dotfileName)
	} else {
		err = addDotfileFile(app, cleanedPath, dotfileName)
	}

	if err != nil {
		fmt.Println(errorStyle.Render("Error!"))
		fmt.Println(err)
		return
	}

	if err := app.GitCommitChanges("Add", dotfileName); err != nil {
		fmt.Println(errorStyle.Render("Error!"))
		fmt.Println("Error committing changes:", err)
		return
	}

	fmt.Println(successStyle.Render("Success!"))
}

func addDotfileDir(app *app.App, cleanedPath, dotfileName string) error {
	storePath := filepath.Join(app.StoragePath, dotfileName)

	if err := app.CopyDirectory(cleanedPath, storePath); err != nil {
		return fmt.Errorf("error copying directory: %w", err)
	}

	return updateConfig(app, cleanedPath, dotfileName)
}

func addDotfileFile(app *app.App, cleanedPath, dotfileName string) error {
	fileName := filepath.Base(cleanedPath)
	storePath := filepath.Join(app.StoragePath, fileName)

	if err := app.CopyFile(cleanedPath, storePath); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return updateConfig(app, cleanedPath, dotfileName)
}

func updateConfig(app *app.App, cleanedPath, dotfileName string) error {
	if err := app.UpdateConfig(dotfileName, cleanedPath); err != nil {
		return fmt.Errorf("error updating config: %w", err)
	}
	return nil
}
