package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// Init initializes a Git repository in the given path and sets the initial branch
func Init(path, branchName string) error {
	// Initialize the repository
	initCmd := exec.Command("git", "-C", path, "init")
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Set the initial branch name
	branchCmd := exec.Command("git", "-C", path, "branch", "-M", branchName)
	if err := branchCmd.Run(); err != nil {
		return fmt.Errorf("failed to set initial branch to %s: %w", branchName, err)
	}

	return nil
}

// CommitChanges commits changes to the Git repository
func CommitChanges(path, commitMessageFormat, dotfileName, action string) error {
	// Check if there are any changes to commit
	statusCmd := exec.Command("git", "-C", path, "status", "--porcelain")
	status, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git status: %w", err)
	}

	if len(status) == 0 {
		// No changes to commit
		return nil
	}

	cmd := exec.Command("git", "-C", path, "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Parse the commit message template
	tmpl, err := template.New("commit").Parse(commitMessageFormat)
	if err != nil {
		return fmt.Errorf("failed to parse commit message template: %w", err)
	}

	// Create a buffer to store the executed template
	var buf bytes.Buffer

	// Execute the template with the dotfile name and action
	err = tmpl.Execute(&buf, struct {
		Dotfile string
		Action  string
	}{
		Dotfile: dotfileName,
		Action:  action,
	})
	if err != nil {
		return fmt.Errorf("failed to execute commit message template: %w", err)
	}

	// Get the formatted commit message
	commitMessage := buf.String()

	cmd = exec.Command("git", "-C", path, "commit", "-m", commitMessage)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// RepoExists checks if a Git repository exists at the given path
func RepoExists(path string) (bool, error) {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func PushChanges(path string) error {
	// Check if a remote origin is configured
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remote origin not configured: %w", err)
	}

	// Push the changes
	pushCmd := exec.Command("git", "-C", path, "push", "origin", "HEAD")
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %w\nOutput: %s", err, output)
	}

	return nil
}
