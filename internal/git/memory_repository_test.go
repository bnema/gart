package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_Init(t *testing.T) {
	repo := NewMemoryRepository("/test")

	err := repo.Init("main")
	assert.NoError(t, err)

	// Verify the working directory is set correctly
	assert.Equal(t, "/test", repo.GetWorkingDirectory())
}

func TestMemoryRepository_InitWithCustomBranch(t *testing.T) {
	repo := NewMemoryRepository("/test")

	err := repo.Init("custom-branch")
	assert.NoError(t, err)
}

func TestMemoryRepository_AddAndCommit(t *testing.T) {
	memRepo := NewMemoryRepository("/test").(*MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Create a test file
	err = memRepo.CreateFile("test.txt", "Hello, World!")
	require.NoError(t, err)

	// Add the file
	err = memRepo.Add("test.txt")
	assert.NoError(t, err)

	// Commit the changes
	err = memRepo.Commit("Add test file")
	assert.NoError(t, err)
}

func TestMemoryRepository_Status(t *testing.T) {
	memRepo := NewMemoryRepository("/test").(*MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Initially, status should be empty
	files, err := memRepo.Status()
	assert.NoError(t, err)
	assert.Empty(t, files)

	// Create a test file
	err = memRepo.CreateFile("test.txt", "Hello, World!")
	require.NoError(t, err)

	// Status should now show the untracked file
	files, err = memRepo.Status()
	assert.NoError(t, err)
	assert.Contains(t, files, "test.txt")
}

func TestMemoryRepository_RemoteOperations(t *testing.T) {
	repo := NewMemoryRepository("/test")
	
	// Initialize the repository
	err := repo.Init("main")
	require.NoError(t, err)

	// Initially, no remote should exist
	hasRemote, err := repo.HasRemote()
	assert.NoError(t, err)
	assert.False(t, hasRemote)

	// Set a remote
	err = repo.SetRemote("origin", "https://example.com/test.git")
	assert.NoError(t, err)

	// Now remote should exist
	hasRemote, err = repo.HasRemote()
	assert.NoError(t, err)
	assert.True(t, hasRemote)
}

func TestMemoryRepository_PushWithoutRemote(t *testing.T) {
	repo := NewMemoryRepository("/test")
	
	// Initialize the repository
	err := repo.Init("main")
	require.NoError(t, err)

	// Push without remote should fail
	err = repo.Push()
	assert.Error(t, err)
	assert.True(t, IsNoRemoteError(err))
}

func TestMemoryRepository_PushWithRemote(t *testing.T) {
	memRepo := NewMemoryRepository("/test").(*MemoryRepository)
	
	// Initialize the repository
	err := memRepo.Init("main")
	require.NoError(t, err)

	// Set a remote
	err = memRepo.SetRemote("origin", "https://example.com/test.git")
	require.NoError(t, err)

	// Create and commit a file
	err = memRepo.CreateFile("test.txt", "Hello, World!")
	require.NoError(t, err)
	
	err = memRepo.Add("test.txt")
	require.NoError(t, err)
	
	err = memRepo.Commit("Add test file")
	require.NoError(t, err)

	// Push should succeed (simulated in memory)
	err = memRepo.Push()
	assert.NoError(t, err)
}

func TestMemoryRepository_OperationsWithoutInit(t *testing.T) {
	repo := NewMemoryRepository("/test")

	// Operations without init should fail
	err := repo.Add("test.txt")
	assert.Error(t, err)
	assert.True(t, IsNotRepositoryError(err))

	err = repo.Commit("test commit")
	assert.Error(t, err)
	assert.True(t, IsNotRepositoryError(err))

	err = repo.Push()
	assert.Error(t, err)
	assert.True(t, IsNotRepositoryError(err))

	_, err = repo.Status()
	assert.Error(t, err)
	assert.True(t, IsNotRepositoryError(err))

	_, err = repo.HasRemote()
	assert.Error(t, err)
	assert.True(t, IsNotRepositoryError(err))
}