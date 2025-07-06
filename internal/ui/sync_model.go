package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/security"
	"github.com/bnema/gart/internal/system"
)

// RunSyncView runs the sync view and returns whether to continue with additional syncs
func RunSyncView(app *app.App, ignores []string) bool {
	sourcePath := app.Dotfile.Path

	// Check if the source is a file or directory
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		fmt.Printf("Error accessing source path: %v\n", err)
		return false
	}

	var storePath string
	if sourceInfo.IsDir() {
		// For directories, use the dotfile name as is
		storePath = filepath.Join(app.StoragePath, app.Dotfile.Name)
	} else {
		// For files, include the file extension in the store path
		ext := filepath.Ext(sourcePath)
		storePath = filepath.Join(app.StoragePath, app.Dotfile.Name+ext)
	}

	// Run security scan before checking for changes
	securityContext := security.NewSecurityContext(app.Config.Settings.Security)

	fmt.Printf("%s\n", scanningStyle.Render(fmt.Sprintf(" Running security scan for '%s'...", app.Dotfile.Name)))
	securityReport, err := securityContext.ScanPath(sourcePath, ignores)
	if err != nil {
		fmt.Printf("Security scan error: %v\n", err)
		return false
	}

	// Handle security findings
	if securityReport.TotalFindings > 0 {
		// Display the security report using UI styling
		DisplaySecurityReport(securityReport)

		proceed, err := securityContext.InteractivePrompt(securityReport)
		if err != nil {
			fmt.Printf("Error handling security prompt: %v\n", err)
			return false
		}

		if !proceed {
			fmt.Printf("Sync aborted due to security concerns.\n")
			return false
		}

		// Add security-based ignores
		securityIgnores := securityContext.GetSecurityIgnores(securityReport)
		ignores = append(ignores, securityIgnores...)
	} else {
		fmt.Printf("%s\n", securityPassStyle.Render("󰸞 Security scan passed - no issues found."))
	}

	// Check for changes before copying
	changed, err := system.DiffFiles(sourcePath, storePath, ignores, app.Config.Settings.ReverseSyncMode)
	if err != nil {
		fmt.Printf("Error comparing dotfiles: %v\n", err)
		return false
	}

	if changed {
		fmt.Print(changedStyle.Render(fmt.Sprintf("Changes detected in '%s'. Updating...", app.Dotfile.Name)))

		var err error
		if sourceInfo.IsDir() {
			// Handle directory
			err = system.CopyDirectory(sourcePath, storePath, ignores)
		} else {
			// Handle single file
			err = os.MkdirAll(filepath.Dir(storePath), 0755)
			if err == nil {
				err = system.CopyFile(sourcePath, storePath, ignores)
			}
		}

		if err != nil {
			fmt.Printf(" %s\n", errorStyle.Render(fmt.Sprintf("Error: %v", err)))
			return false
		}

		// Only commit changes if we're in push mode (not reverse sync)
		if !app.Config.Settings.ReverseSyncMode {
			if err := app.GitCommitChanges("Update", app.Dotfile.Name); err != nil {
				fmt.Printf(" %s\n", errorStyle.Render(fmt.Sprintf("Error committing changes: %v", err)))
				return false
			}
		}

		fmt.Printf(" %s\n", successStyle.Render("Success!"))
	} else {
		fmt.Println(unchangedStyle.Render(fmt.Sprintf("No changes detected in '%s' since the last update.", app.Dotfile.Name)))
	}
	return true
}
