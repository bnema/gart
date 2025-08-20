package git

import (
	"errors"
	"fmt"
)

// Common git operation errors
var (
	ErrNoRemote    = errors.New("no remote origin configured")
	ErrAuthFailed  = errors.New("authentication failed")
	ErrNotRepository = errors.New("not a git repository")
)

// GitError represents a git operation error with context
type GitError struct {
	Op   string // Operation that failed (e.g., "init", "add", "commit", "push")
	Path string // Path where the operation failed
	Err  error  // Underlying error
}

func (e *GitError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("git %s in %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("git %s: %v", e.Op, e.Err)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// IsNoRemoteError checks if an error is due to missing remote configuration
func IsNoRemoteError(err error) bool {
	return errors.Is(err, ErrNoRemote)
}

// IsAuthError checks if an error is due to authentication failure
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthFailed)
}

// IsNotRepositoryError checks if an error is due to not being in a git repository
func IsNotRepositoryError(err error) bool {
	return errors.Is(err, ErrNotRepository)
}