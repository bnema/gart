package git

import "context"

// GitRepository defines the interface for git operations
// This interface allows for easy testing by providing mockable methods
type GitRepository interface {
	// Init initializes a git repository with the specified branch name
	Init(branch string) error

	// Add stages files for commit. Use "." to add all files
	Add(patterns ...string) error

	// Commit creates a commit with the given message
	Commit(message string) error

	// Push pushes commits to the remote repository
	Push() error

	// PushContext pushes commits to the remote repository with context for cancellation
	PushContext(ctx context.Context) error

	// Status returns a list of changed files
	Status() ([]string, error)

	// HasRemote checks if a remote named "origin" is configured
	HasRemote() (bool, error)

	// SetRemote sets a remote repository URL
	SetRemote(name, url string) error

	// GetWorkingDirectory returns the path to the working directory
	GetWorkingDirectory() string

	// Exists checks if a git repository exists at the working directory
	Exists() (bool, error)
}