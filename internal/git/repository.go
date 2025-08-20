package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
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

	// Create filesystem and storage
	fs := osfs.New(r.workingDir)
	storage := filesystem.NewStorage(fs, nil)

	// Initialize the repository
	repo, err := git.Init(storage, fs)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	r.repo = repo

	// Set the initial branch name if specified
	if branch != "" {
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/"+branch))
		if err := r.repo.Storer.SetReference(headRef); err != nil {
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

	fs := osfs.New(r.workingDir)
	storage := filesystem.NewStorage(fs, nil)

	repo, err := git.Open(storage, fs)
	if err != nil {
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
		_, err := worktree.Add(pattern)
		if err != nil {
			return fmt.Errorf("failed to add pattern %s: %w", pattern, err)
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

	err = r.repo.PushContext(ctx, &git.PushOptions{
		RemoteName: "origin",
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

// HasRemote checks if a remote named "origin" is configured
func (r *Repository) HasRemote() (bool, error) {
	if err := r.openRepository(); err != nil {
		return false, err
	}

	remotes, err := r.repo.Remotes()
	if err != nil {
		return false, fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return true, nil
		}
	}

	return false, nil
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