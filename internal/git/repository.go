package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// Repository implements GitRepository using go-git library
type Repository struct {
	workingDir string
	repo       *git.Repository
}

// NewRepository creates a new Repository instance
func NewRepository(workingDir string) (GitRepository, error) {
	absPath, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Repository{
		workingDir: absPath,
	}, nil
}

// Init initializes a git repository with the specified branch name
func (r *Repository) Init(branch string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(r.workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if repository already exists
	if exists, err := r.Exists(); err != nil {
		return fmt.Errorf("failed to check if repository exists: %w", err)
	} else if exists {
		// Repository already exists, just open it
		repo, err := git.PlainOpen(r.workingDir)
		if err != nil {
			return fmt.Errorf("failed to open existing repository: %w", err)
		}
		r.repo = repo
		return nil
	}

	// Initialize new repository using PlainInit
	repo, err := git.PlainInit(r.workingDir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	r.repo = repo

	// Set the initial branch name if specified and different from default
	if branch != "" && branch != "master" {
		// Set HEAD to point to the desired branch before any commits are made
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/"+branch))
		if err := repo.Storer.SetReference(headRef); err != nil {
			return fmt.Errorf("failed to set initial branch to %s: %w", branch, err)
		}
	}

	return nil
}

// openRepository opens an existing repository if not already opened
func (r *Repository) openRepository() error {
	if r.repo != nil {
		return nil
	}

	// Use PlainOpen for reliable repository opening
	repo, err := git.PlainOpen(r.workingDir)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return &GitError{
				Op:   "open",
				Path: r.workingDir,
				Err:  ErrNotRepository,
			}
		}
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	r.repo = repo
	return nil
}

// Add stages files for commit
func (r *Repository) Add(patterns ...string) error {
	if err := r.openRepository(); err != nil {
		return err
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// If no patterns specified, add all files
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	for _, pattern := range patterns {
		// Check if pattern contains wildcards, use AddGlob for patterns
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") || strings.Contains(pattern, "[") {
			err := worktree.AddGlob(pattern)
			if err != nil {
				return fmt.Errorf("failed to add glob pattern %s: %w", pattern, err)
			}
		} else {
			// Use regular Add for exact paths
			_, err := worktree.Add(pattern)
			if err != nil {
				return fmt.Errorf("failed to add path %s: %w", pattern, err)
			}
		}
	}

	return nil
}

// Commit creates a commit with the given message
func (r *Repository) Commit(message string) error {
	if err := r.openRepository(); err != nil {
		return err
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if there are any changes to commit
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		// No changes to commit
		return nil
	}

	// Create commit
	_, err = worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Gart",
			Email: "gart@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes commits to the remote repository
func (r *Repository) Push() error {
	return r.PushContext(context.Background())
}

// PushContext pushes commits to the remote repository with context
func (r *Repository) PushContext(ctx context.Context) error {
	if err := r.openRepository(); err != nil {
		return err
	}

	// Check if remote exists
	hasRemote, err := r.HasRemote()
	if err != nil {
		return fmt.Errorf("failed to check remote: %w", err)
	}

	if !hasRemote {
		return &GitError{
			Op:   "push",
			Path: r.workingDir,
			Err:  ErrNoRemote,
		}
	}

	// Get the first available remote
	remotes, err := r.repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	if len(remotes) == 0 {
		return &GitError{
			Op:   "push",
			Path: r.workingDir,
			Err:  ErrNoRemote,
		}
	}

	// Use the first available remote
	remoteName := remotes[0].Config().Name
	remoteConfig := remotes[0].Config()
	
	// Get authentication for the remote URL
	var auth transport.AuthMethod
	if len(remoteConfig.URLs) > 0 {
		auth, err = r.getAuthMethod(remoteConfig.URLs[0])
		if err != nil {
			// Log the auth error but continue without auth (might work for public repos)
			fmt.Printf("Warning: Git authentication failed, continuing without auth: %v\n", err)
		}
	}

	err = r.repo.PushContext(ctx, &git.PushOptions{
		RemoteName: remoteName,
		Auth:       auth,
	})
	if err != nil {
		// Check if it's a "no changes" error, which is not actually an error
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// Status returns a list of changed files
func (r *Repository) Status() ([]string, error) {
	if err := r.openRepository(); err != nil {
		return nil, err
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var files []string
	for file := range status {
		files = append(files, file)
	}

	return files, nil
}

// HasRemote checks if any remote is configured
func (r *Repository) HasRemote() (bool, error) {
	if err := r.openRepository(); err != nil {
		return false, err
	}

	remotes, err := r.repo.Remotes()
	if err != nil {
		return false, fmt.Errorf("failed to get remotes: %w", err)
	}

	return len(remotes) > 0, nil
}

// SetRemote sets a remote repository URL
func (r *Repository) SetRemote(name, url string) error {
	if err := r.openRepository(); err != nil {
		return err
	}

	// Create remote config
	_, err := r.repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		return fmt.Errorf("failed to create remote %s: %w", name, err)
	}

	return nil
}

// GetWorkingDirectory returns the path to the working directory
func (r *Repository) GetWorkingDirectory() string {
	return r.workingDir
}

// Exists checks if a git repository exists at the working directory
func (r *Repository) Exists() (bool, error) {
	gitDir := filepath.Join(r.workingDir, ".git")
	_, err := os.Stat(gitDir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// getAuthMethod attempts to get appropriate authentication for git operations
func (r *Repository) getAuthMethod(remoteURL string) (transport.AuthMethod, error) {
	if strings.HasPrefix(remoteURL, "https://") {
		// Try to get credentials from git config or environment
		return r.getHTTPSAuth()
	} else if strings.HasPrefix(remoteURL, "git@") || strings.Contains(remoteURL, "ssh://") {
		// SSH authentication
		return r.getSSHAuth()
	}
	
	// No authentication needed for local or other protocols
	return nil, nil
}

// getSSHAuth attempts to authenticate using SSH agent first, then SSH keys
func (r *Repository) getSSHAuth() (transport.AuthMethod, error) {
	// Try SSH agent first (most common and convenient)
	sshAgent, err := ssh.NewSSHAgentAuth("git")
	if err == nil {
		return sshAgent, nil
	}
	
	// SSH agent failed, try SSH keys from standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	// Try different SSH key files in order of preference
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
	}
	
	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			// Try without passphrase first
			publicKeys, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
			if err == nil {
				return publicKeys, nil
			}
			
			// If that fails, the key probably has a passphrase
			// For now, we'll skip passphrase-protected keys
			// TODO: Implement interactive passphrase prompting
		}
	}
	
	return nil, fmt.Errorf("no suitable SSH authentication method found")
}

// getHTTPSAuth attempts to get HTTPS authentication from git config or environment
func (r *Repository) getHTTPSAuth() (transport.AuthMethod, error) {
	// Try to get credentials from environment variables
	if token := os.Getenv("GIT_TOKEN"); token != "" {
		return &http.BasicAuth{
			Username: "token", // GitHub personal access token format
			Password: token,
		}, nil
	}
	
	if username := os.Getenv("GIT_USERNAME"); username != "" {
		password := os.Getenv("GIT_PASSWORD")
		return &http.BasicAuth{
			Username: username,
			Password: password,
		}, nil
	}
	
	// TODO: Read from .gitconfig or credential helpers
	// For now, return nil to attempt unauthenticated access
	return nil, nil
}

// getRemoteURL gets the URL of the first available remote
func (r *Repository) getRemoteURL() (string, error) {
	if err := r.openRepository(); err != nil {
		return "", err
	}
	
	remotes, err := r.repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}
	
	if len(remotes) == 0 {
		return "", fmt.Errorf("no remotes configured")
	}
	
	// Get the first URL from the first remote
	config := remotes[0].Config()
	if len(config.URLs) == 0 {
		return "", fmt.Errorf("remote has no URLs")
	}
	
	return config.URLs[0], nil
}