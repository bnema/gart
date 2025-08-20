package app

import (
	"testing"

	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/git"
	"github.com/bnema/gart/internal/git/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestApp_GitCommitChanges_AutoPushEnabled_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	// Create app with git versioning and auto_push enabled
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
		gitRepo: mockRepo,
	}

	// Set expectations
	mockRepo.EXPECT().Add(".").Return(nil)
	mockRepo.EXPECT().Commit("Add test.txt").Return(nil)
	mockRepo.EXPECT().Push().Return(nil).Times(1) // auto_push enabled, should be called

	err := app.GitCommitChanges("Add", "test.txt")
	assert.NoError(t, err)
}

func TestApp_GitCommitChanges_AutoPushEnabled_PushFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
		gitRepo: mockRepo,
	}

	pushError := &git.GitError{Op: "push", Err: git.ErrNoRemote}

	// Set expectations
	mockRepo.EXPECT().Add(".").Return(nil)
	mockRepo.EXPECT().Commit("Update config.yaml").Return(nil)
	mockRepo.EXPECT().Push().Return(pushError).Times(1)

	err := app.GitCommitChanges("Update", "config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push changes")
	assert.Contains(t, err.Error(), "no remote origin configured")
}

func TestApp_GitCommitChanges_AutoPushDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            false, // auto_push disabled
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
		gitRepo: mockRepo,
	}

	// Set expectations - Push should NOT be called
	mockRepo.EXPECT().Add(".").Return(nil)
	mockRepo.EXPECT().Commit("Remove old.txt").Return(nil)
	// Note: No Push() expectation - it should not be called

	err := app.GitCommitChanges("Remove", "old.txt")
	assert.NoError(t, err)
}

func TestApp_GitCommitChanges_GitVersioningDisabled(t *testing.T) {
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: false, // git versioning disabled
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
	}

	// No git operations should be performed
	err := app.GitCommitChanges("Add", "test.txt")
	assert.NoError(t, err)
}

func TestApp_GitCommitChanges_AddFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
		gitRepo: mockRepo,
	}

	addError := &git.GitError{Op: "add", Err: git.ErrNotRepository}

	// Set expectations - Add fails, so Commit and Push should not be called
	mockRepo.EXPECT().Add(".").Return(addError)

	err := app.GitCommitChanges("Add", "test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add changes")
}

func TestApp_GitCommitChanges_CommitFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Action }} {{ .Dotfile }}",
				},
			},
		},
		gitRepo: mockRepo,
	}

	commitError := &git.GitError{Op: "commit", Err: git.ErrNotRepository}

	// Set expectations - Commit fails, so Push should not be called
	mockRepo.EXPECT().Add(".").Return(nil)
	mockRepo.EXPECT().Commit("Add test.txt").Return(commitError)

	err := app.GitCommitChanges("Add", "test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit changes")
}

func TestApp_GitCommitChanges_InvalidTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            true,
					CommitMessageFormat: "{{ .Invalid }}", // Invalid template
				},
			},
		},
		gitRepo: mockRepo,
	}

	// Set expectations - Add is called but then template parsing fails
	mockRepo.EXPECT().Add(".").Return(nil)

	err := app.GitCommitChanges("Add", "test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute commit message template")
}

func TestApp_GitCommitChanges_ComplexTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{
		Config: &config.Config{
			Settings: config.SettingsConfig{
				GitVersioning: true,
				Git: config.GitConfig{
					AutoPush:            false,
					CommitMessageFormat: "feat({{ .Dotfile }}): {{ .Action }} configuration file",
				},
			},
		},
		gitRepo: mockRepo,
	}

	// Set expectations with the expected commit message
	mockRepo.EXPECT().Add(".").Return(nil)
	mockRepo.EXPECT().Commit("feat(nvim): ADD configuration file").Return(nil)

	err := app.GitCommitChanges("ADD", "nvim")
	assert.NoError(t, err)
}

func TestApp_getOrCreateGitRepository(t *testing.T) {
	app := &App{
		StoragePath: "/test/path",
	}

	// First call should create a new repository
	repo1, err := app.getOrCreateGitRepository()
	require.NoError(t, err)
	assert.NotNil(t, repo1)
	assert.Equal(t, "/test/path", repo1.GetWorkingDirectory())

	// Second call should return the same repository
	repo2, err := app.getOrCreateGitRepository()
	require.NoError(t, err)
	assert.Same(t, repo1, repo2)
}

func TestApp_SetGitRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockGitRepository(ctrl)
	
	app := &App{}

	app.SetGitRepository(mockRepo)

	// Verify the repository was set
	repo, err := app.getOrCreateGitRepository()
	require.NoError(t, err)
	assert.Same(t, mockRepo, repo)
}