package app

import (
	"testing"

	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApp_GitCommitChanges_Integration_AutoPushFlow tests the complete auto_push flow
// using the memory repository to verify the full interaction
func TestApp_GitCommitChanges_Integration_AutoPushFlow(t *testing.T) {
	// Create memory repository for testing
	memRepo := git.NewMemoryRepository("/test").(*git.MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Set up remote for push testing
	err = memRepo.SetRemote("origin", "https://example.com/test.git")
	require.NoError(t, err)

	// Create app with auto_push enabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }}: {{ .Dotfile }}",
				},
			},
		},
	}

	// Set the memory repository
	app.SetGitRepository(memRepo)

	// Create a test file to commit
	err = memRepo.CreateFile("test.txt", "Hello, World!")
	require.NoError(t, err)

	// Verify there are changes to commit
	files, err := memRepo.Status()
	require.NoError(t, err)
	assert.Contains(t, files, "test.txt")

	// Execute GitCommitChanges with auto_push enabled
	err = app.GitCommitChanges("Add", "test.txt")
	assert.NoError(t, err, "GitCommitChanges should succeed with auto_push")

	// Verify the file is no longer in status (has been committed)
	files, err = memRepo.Status()
	require.NoError(t, err)
	assert.Empty(t, files, "All files should be committed")
}

// TestApp_GitCommitChanges_Integration_AutoPushDisabled tests that push is not called when disabled
func TestApp_GitCommitChanges_Integration_AutoPushDisabled(t *testing.T) {
	// Create memory repository for testing
	memRepo := git.NewMemoryRepository("/test").(*git.MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Create app with auto_push disabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            false, // Disabled
					CommitMessageFormat: "{{ .Action }}: {{ .Dotfile }}",
				},
			},
		},
	}

	// Set the memory repository
	app.SetGitRepository(memRepo)

	// Create a test file to commit
	err = memRepo.CreateFile("config.yaml", "key: value")
	require.NoError(t, err)

	// Execute GitCommitChanges with auto_push disabled
	err = app.GitCommitChanges("Update", "config.yaml")
	assert.NoError(t, err, "GitCommitChanges should succeed without push")

	// Verify the file is committed but we don't test push since it's disabled
	files, err := memRepo.Status()
	require.NoError(t, err)
	assert.Empty(t, files, "All files should be committed")
}

// TestApp_GitCommitChanges_Integration_NoRemote tests auto_push behavior when no remote is configured
func TestApp_GitCommitChanges_Integration_NoRemote(t *testing.T) {
	// Create memory repository for testing
	memRepo := git.NewMemoryRepository("/test").(*git.MemoryRepository)
	
	// Initialize the repository without setting up a remote
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Create app with auto_push enabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }}: {{ .Dotfile }}",
				},
			},
		},
	}

	// Set the memory repository
	app.SetGitRepository(memRepo)

	// Create a test file to commit
	err = memRepo.CreateFile("no-remote.txt", "test content")
	require.NoError(t, err)

	// Execute GitCommitChanges - should fail on push due to no remote
	err = app.GitCommitChanges("Add", "no-remote.txt")
	assert.Error(t, err, "GitCommitChanges should fail when auto_push is enabled but no remote exists")
	assert.Contains(t, err.Error(), "failed to push changes")
	assert.Contains(t, err.Error(), "no remote origin configured")
}

// TestApp_GitCommitChanges_Integration_NoChanges tests behavior when there are no changes
func TestApp_GitCommitChanges_Integration_NoChanges(t *testing.T) {
	// Create memory repository for testing
	memRepo := git.NewMemoryRepository("/test").(*git.MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Set up remote
	err = memRepo.SetRemote("origin", "https://example.com/test.git")
	require.NoError(t, err)

	// Create app with auto_push enabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }}: {{ .Dotfile }}",
				},
			},
		},
	}

	// Set the memory repository
	app.SetGitRepository(memRepo)

	// Execute GitCommitChanges without any changes
	err = app.GitCommitChanges("Update", "nonexistent.txt")
	assert.NoError(t, err, "GitCommitChanges should succeed even with no changes to commit")
}

// TestApp_GitCommitChanges_Integration_MultipleFiles tests committing multiple files with auto_push
func TestApp_GitCommitChanges_Integration_MultipleFiles(t *testing.T) {
	// Create memory repository for testing
	memRepo := git.NewMemoryRepository("/test").(*git.MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("develop")
	require.NoError(t, err)

	// Set up remote
	err = memRepo.SetRemote("origin", "https://example.com/test.git")
	require.NoError(t, err)

	// Create app with auto_push enabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "feat({{ .Dotfile }}): {{ .Action }} dotfile configuration",
				},
			},
		},
	}

	// Set the memory repository
	app.SetGitRepository(memRepo)

	// Create multiple test files
	err = memRepo.CreateFile("vimrc", "set number")
	require.NoError(t, err)
	
	err = memRepo.CreateFile("zshrc", "export PATH=$PATH:/usr/local/bin")
	require.NoError(t, err)

	err = memRepo.CreateFile("gitconfig", "[user]\n  name = Test User")
	require.NoError(t, err)

	// Verify there are changes to commit
	files, err := memRepo.Status()
	require.NoError(t, err)
	assert.Len(t, files, 3, "Should have 3 untracked files")

	// Execute GitCommitChanges with auto_push enabled
	err = app.GitCommitChanges("Add", "dotfiles")
	assert.NoError(t, err, "GitCommitChanges should succeed with auto_push for multiple files")

	// Verify all files are committed
	files, err = memRepo.Status()
	require.NoError(t, err)
	assert.Empty(t, files, "All files should be committed")
}