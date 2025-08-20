package git

import (
	"context"
	"fmt"
	"log"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// MemoryRepository implements GitRepository using in-memory storage
// This is perfect for testing as it doesn't touch the filesystem
type MemoryRepository struct {
	workingDir string
	repo       *git.Repository
	fs         billy.Filesystem
	storage    *memory.Storage
}

// NewMemoryRepository creates a new in-memory repository for testing
func NewMemoryRepository(workingDir string) GitRepository {
	fs := memfs.New()
	storage := memory.NewStorage()

	return &MemoryRepository{
		workingDir: workingDir,
		fs:         fs,
		storage:    storage,
	}
}

// Init initializes a git repository with the specified branch name
func (r *MemoryRepository) Init(branch string) error {
	// Initialize the repository
	repo, err := git.Init(r.storage, r.fs)
	if err != nil {
		return fmt.Errorf("failed to initialize in-memory git repository: %w", err)
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

// Add stages files for commit
func (r *MemoryRepository) Add(patterns ...string) error {
	if r.repo == nil {
		return &GitError{
			Op:   "add",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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
func (r *MemoryRepository) Commit(message string) error {
	if r.repo == nil {
		return &GitError{
			Op:   "commit",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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
			Name:  "Gart Test",
			Email: "test@localhost",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes commits to the remote repository
func (r *MemoryRepository) Push() error {
	return r.PushContext(context.Background())
}

// PushContext pushes commits to the remote repository with context
func (r *MemoryRepository) PushContext(ctx context.Context) error {
	if r.repo == nil {
		return &GitError{
			Op:   "push",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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

	// For in-memory repositories, we'll simulate a successful push
	// In real tests, you might want to set up a mock remote or use git.NoErrAlreadyUpToDate
	return nil
}

// Status returns a list of changed files
func (r *MemoryRepository) Status() ([]string, error) {
	if r.repo == nil {
		return nil, &GitError{
			Op:   "status",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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
func (r *MemoryRepository) HasRemote() (bool, error) {
	if r.repo == nil {
		return false, &GitError{
			Op:   "remote",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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
func (r *MemoryRepository) SetRemote(name, url string) error {
	if r.repo == nil {
		return &GitError{
			Op:   "remote",
			Path: r.workingDir,
			Err:  ErrNotRepository,
		}
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
func (r *MemoryRepository) GetWorkingDirectory() string {
	return r.workingDir
}

// Exists checks if a git repository exists (always true for memory repos after Init)
func (r *MemoryRepository) Exists() (bool, error) {
	return r.repo != nil, nil
}

// CreateFile creates a file in the in-memory filesystem for testing
func (r *MemoryRepository) CreateFile(filename, content string) error {
	file, err := r.fs.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file %s: %v", filename, closeErr)
		}
	}()

	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write content to file %s: %w", filename, err)
	}

	return nil
}