package ui

import (
	"fmt"
	"path/filepath"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/security"
)

func RunAddDotfileView(app *app.App, path string, dotfileName string, ignores []string) {
	path = app.ExpandHomeDir(path)
	cleanedPath := filepath.Clean(path)

	fmt.Printf("Adding dotfile %s... ", dotfileName)

	// Create security context and scan path
	securityConfig := security.DefaultSecurityConfig()
	securityCtx := security.NewSecurityContext(securityConfig)

	// Scan for security issues before adding
	report, err := securityCtx.ScanPath(cleanedPath, ignores)
	if err != nil {
		fmt.Println(errorStyle.Render("Security scan failed!"))
		fmt.Println("Error:", err)
		return
	}

	// Handle security findings interactively
	if report.TotalFindings > 0 {
		DisplaySecurityFindings(report)

		proceed, _, err := securityCtx.InteractivePrompt(report)
		if err != nil {
			fmt.Println(errorStyle.Render("Error!"))
			fmt.Println("Security check failed:", err)
			return
		}
		if !proceed {
			fmt.Println("⚠️  Add operation cancelled due to security concerns")
			return
		}

		// Note: For add command, we don't use skipAll flag since it's a single operation
		// Security findings don't affect permanent configuration
	}

	var addErr error
	if app.IsDir(path) {
		addErr = addDotfileDir(app, cleanedPath, dotfileName, ignores)
	} else {
		addErr = addDotfileFile(app, cleanedPath, dotfileName, ignores)
	}

	if addErr != nil {
		fmt.Println(errorStyle.Render("Error!"))
		fmt.Println(addErr)
		return
	}

	// Create commit message with security status
	commitMsg := fmt.Sprintf("Add %s", dotfileName)
	if report.TotalFindings > 0 {
		commitMsg += fmt.Sprintf(" (security: %d findings, risk: %s)", report.TotalFindings, report.HighestRisk)
	} else {
		commitMsg += " (security: clean)"
	}

	if err := app.GitCommitChanges("Add", commitMsg); err != nil {
		fmt.Println(errorStyle.Render("Error!"))
		fmt.Println("Error committing changes:", err)
		return
	}

	fmt.Println(successStyle.Render("Success!"))
}

func addDotfileDir(app *app.App, cleanedPath, dotfileName string, ignores []string) error {
	storePath := filepath.Join(app.StoragePath, dotfileName)

	if err := app.CopyDirectory(cleanedPath, storePath, ignores); err != nil {
		return fmt.Errorf("error copying directory: %w", err)
	}

	return updateConfig(app, cleanedPath, dotfileName, ignores)
}

func addDotfileFile(app *app.App, cleanedPath, dotfileName string, ignores []string) error {
	fileName := filepath.Base(cleanedPath)
	storePath := filepath.Join(app.StoragePath, fileName)

	if err := app.CopyFile(cleanedPath, storePath, ignores); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return updateConfig(app, cleanedPath, dotfileName, ignores)
}

func updateConfig(app *app.App, cleanedPath, dotfileName string, ignores []string) error {
	if err := app.UpdateConfig(dotfileName, cleanedPath, ignores); err != nil {
		return fmt.Errorf("error updating config: %w", err)
	}
	return nil
}
