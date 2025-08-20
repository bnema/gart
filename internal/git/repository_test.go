package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Init_NewRepository(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	assert.NoError(t, err)

	// Verify .git directory exists
	gitDir := filepath.Join(tmpDir, ".git")
	_, err = os.Stat(gitDir)
	assert.NoError(t, err)

	// Verify repository exists
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRepository_Init_ExistingRepository(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repo first time
	repo1, err := NewRepository(tmpDir)
	require.NoError(t, err)
	err = repo1.Init("main")
	require.NoError(t, err)

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = repo1.Add(".")
	require.NoError(t, err)
	err = repo1.Commit("initial commit")
	require.NoError(t, err)

	// Create new Repository instance and init again
	repo2, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Should not fail when repository already exists
	err = repo2.Init("main")
	assert.NoError(t, err)

	// Should be able to access existing commit
	status, err := repo2.Status()
	assert.NoError(t, err)
	assert.Empty(t, status) // No changes, file was committed
}

func TestRepository_Init_WithCustomBranch(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Create with custom branch
	err = repo.Init("develop")
	assert.NoError(t, err)

	// Verify repository exists
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.True(t, exists)

	// Add a file to make the branch active
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = repo.Add(".")
	assert.NoError(t, err)
	err = repo.Commit("initial commit")
	assert.NoError(t, err)

	// Verify branch name (this requires opening the repo)
	r := repo.(*Repository)
	head, err := r.repo.Head()
	assert.NoError(t, err)
	assert.Equal(t, "develop", head.Name().Short())
}

func TestRepository_OpenRepository_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repo first
	repo1, err := NewRepository(tmpDir)
	require.NoError(t, err)
	err = repo1.Init("main")
	require.NoError(t, err)

	// Create new Repository instance
	repo2, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Should be able to open existing repo
	r := repo2.(*Repository)
	err = r.openRepository()
	assert.NoError(t, err)
	assert.NotNil(t, r.repo)
}

func TestRepository_OpenRepository_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Try to open non-existent repository
	r := repo.(*Repository)
	err = r.openRepository()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestRepository_Add_AllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	err = os.WriteFile(testFile1, []byte("test content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	require.NoError(t, err)

	// Add all files
	err = repo.Add(".")
	assert.NoError(t, err)

	// Check status - files should be staged
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "test1.txt")
	assert.Contains(t, files, "test2.txt")
}

func TestRepository_Add_SpecificPattern(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.log")
	err = os.WriteFile(testFile1, []byte("test content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	require.NoError(t, err)

	// Add only .txt files
	err = repo.Add("*.txt")
	assert.NoError(t, err)

	// Check status - only .txt file should be staged
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 2) // One staged, one untracked
	assert.Contains(t, files, "test1.txt")
	assert.Contains(t, files, "test2.log")
}

func TestRepository_Commit_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create and add test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = repo.Add(".")
	require.NoError(t, err)

	// Commit changes
	err = repo.Commit("test commit")
	assert.NoError(t, err)

	// Check status after commit - should be clean
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Empty(t, files)
}

func TestRepository_Commit_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Try to commit with no changes
	err = repo.Commit("empty commit")
	assert.NoError(t, err) // Should not error, just return early
}

func TestRepository_Status_Clean(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Status should be clean initially
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Empty(t, files)
}

func TestRepository_Status_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	err = os.WriteFile(testFile1, []byte("test content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	require.NoError(t, err)

	// Status should show untracked files
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "test1.txt")
	assert.Contains(t, files, "test2.txt")

	// Add one file
	err = repo.Add("test1.txt")
	require.NoError(t, err)

	// Status should show staged and untracked
	files, err = repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "test1.txt")
	assert.Contains(t, files, "test2.txt")
}

func TestRepository_Exists_True(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Before init
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.False(t, exists)

	// After init
	err = repo.Init("main")
	require.NoError(t, err)

	exists, err = repo.Exists()
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRepository_Exists_False(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// No .git directory
	exists, err := repo.Exists()
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_HasRemote_False(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	hasRemote, err := repo.HasRemote()
	assert.NoError(t, err)
	assert.False(t, hasRemote)
}

func TestRepository_SetRemote_Success(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Set remote
	err = repo.SetRemote("origin", "https://github.com/user/repo.git")
	assert.NoError(t, err)

	// Check if remote exists
	hasRemote, err := repo.HasRemote()
	assert.NoError(t, err)
	assert.True(t, hasRemote)
}

func TestRepository_Push_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = repo.Add(".")
	require.NoError(t, err)
	err = repo.Commit("initial commit")
	require.NoError(t, err)

	// Try to push without remote
	err = repo.Push()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no remote origin configured")
}

func TestRepository_GetWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	workDir := repo.GetWorkingDirectory()
	absPath, _ := filepath.Abs(tmpDir)
	assert.Equal(t, absPath, workDir)
}

func TestNewRepository_RelativePath(t *testing.T) {
	// Use relative path
	repo, err := NewRepository("./test")
	require.NoError(t, err)

	// Should convert to absolute path
	workDir := repo.GetWorkingDirectory()
	assert.True(t, filepath.IsAbs(workDir))
	assert.Contains(t, workDir, "test")
}

func TestRepository_MultipleOperations_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repository
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create multiple files
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		err = os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0644)
		require.NoError(t, err)
	}

	// Add all files
	err = repo.Add(".")
	assert.NoError(t, err)

	// Check status
	files, err := repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 3)

	// Commit
	err = repo.Commit("add multiple files")
	assert.NoError(t, err)

	// Status should be clean
	files, err = repo.Status()
	assert.NoError(t, err)
	assert.Empty(t, files)

	// Modify one file
	testFile := filepath.Join(tmpDir, "file0.txt")
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Status should show modified file
	files, err = repo.Status()
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "file0.txt")

	// Add and commit the change
	err = repo.Add("file0.txt")
	assert.NoError(t, err)

	err = repo.Commit("modify file0")
	assert.NoError(t, err)

	// Status should be clean again
	files, err = repo.Status()
	assert.NoError(t, err)
	assert.Empty(t, files)
}

// Authentication Tests

func TestRepository_getAuthMethod_HTTPSWithToken(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Set environment variable for test
	originalToken := os.Getenv("GIT_TOKEN")
	defer func() {
		if originalToken != "" {
			if err := os.Setenv("GIT_TOKEN", originalToken); err != nil {
				t.Errorf("Failed to restore GIT_TOKEN: %v", err)
			}
		} else {
			if err := os.Unsetenv("GIT_TOKEN"); err != nil {
				t.Errorf("Failed to unset GIT_TOKEN: %v", err)
			}
		}
	}()

	if err := os.Setenv("GIT_TOKEN", "test-token-123"); err != nil {
		t.Fatalf("Failed to set GIT_TOKEN: %v", err)
	}

	r := repo.(*Repository)
	auth, err := r.getAuthMethod("https://github.com/user/repo.git")
	assert.NoError(t, err)
	assert.NotNil(t, auth)
}

func TestRepository_getAuthMethod_HTTPSWithCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Set environment variables for test
	originalUsername := os.Getenv("GIT_USERNAME")
	originalPassword := os.Getenv("GIT_PASSWORD")
	defer func() {
		if originalUsername != "" {
			if err := os.Setenv("GIT_USERNAME", originalUsername); err != nil {
				t.Errorf("Failed to restore GIT_USERNAME: %v", err)
			}
		} else {
			if err := os.Unsetenv("GIT_USERNAME"); err != nil {
				t.Errorf("Failed to unset GIT_USERNAME: %v", err)
			}
		}
		if originalPassword != "" {
			if err := os.Setenv("GIT_PASSWORD", originalPassword); err != nil {
				t.Errorf("Failed to restore GIT_PASSWORD: %v", err)
			}
		} else {
			if err := os.Unsetenv("GIT_PASSWORD"); err != nil {
				t.Errorf("Failed to unset GIT_PASSWORD: %v", err)
			}
		}
	}()

	if err := os.Setenv("GIT_USERNAME", "testuser"); err != nil {
		t.Fatalf("Failed to set GIT_USERNAME: %v", err)
	}
	if err := os.Setenv("GIT_PASSWORD", "testpass"); err != nil {
		t.Fatalf("Failed to set GIT_PASSWORD: %v", err)
	}

	r := repo.(*Repository)
	auth, err := r.getAuthMethod("https://github.com/user/repo.git")
	assert.NoError(t, err)
	assert.NotNil(t, auth)
}

func TestRepository_getAuthMethod_HTTPSNoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	// Clear environment variables for test
	originalToken := os.Getenv("GIT_TOKEN")
	originalUsername := os.Getenv("GIT_USERNAME")
	originalPassword := os.Getenv("GIT_PASSWORD")
	defer func() {
		if originalToken != "" {
			if err := os.Setenv("GIT_TOKEN", originalToken); err != nil {
				t.Errorf("Failed to restore GIT_TOKEN: %v", err)
			}
		}
		if originalUsername != "" {
			if err := os.Setenv("GIT_USERNAME", originalUsername); err != nil {
				t.Errorf("Failed to restore GIT_USERNAME: %v", err)
			}
		}
		if originalPassword != "" {
			if err := os.Setenv("GIT_PASSWORD", originalPassword); err != nil {
				t.Errorf("Failed to restore GIT_PASSWORD: %v", err)
			}
		}
	}()

	if err := os.Unsetenv("GIT_TOKEN"); err != nil {
		t.Errorf("Failed to unset GIT_TOKEN: %v", err)
	}
	if err := os.Unsetenv("GIT_USERNAME"); err != nil {
		t.Errorf("Failed to unset GIT_USERNAME: %v", err)
	}
	if err := os.Unsetenv("GIT_PASSWORD"); err != nil {
		t.Errorf("Failed to unset GIT_PASSWORD: %v", err)
	}

	r := repo.(*Repository)
	auth, err := r.getAuthMethod("https://github.com/user/repo.git")
	assert.NoError(t, err)
	assert.Nil(t, auth) // Should return nil for unauthenticated access
}

func TestRepository_getAuthMethod_SSH(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	r := repo.(*Repository)
	
	// SSH auth will either succeed (SSH agent available) or fail (no SSH keys)
	// Both are valid outcomes for this test
	auth, err := r.getAuthMethod("git@github.com:user/repo.git")
	
	// The method should not return both nil auth and nil error
	if err != nil {
		// If there's an error, auth should be nil
		assert.Nil(t, auth)
		assert.Contains(t, err.Error(), "no suitable SSH authentication method found")
	} else {
		// If no error, auth should not be nil
		assert.NotNil(t, auth)
	}
}

func TestRepository_getAuthMethod_LocalProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	r := repo.(*Repository)
	auth, err := r.getAuthMethod("/local/path/to/repo")
	assert.NoError(t, err)
	assert.Nil(t, auth) // No auth needed for local protocols
}

func TestRepository_getRemoteURL_NoRemotes(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	r := repo.(*Repository)
	url, err := r.getRemoteURL()
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "no remotes configured")
}

func TestRepository_getRemoteURL_WithRemote(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Add a remote
	err = repo.SetRemote("origin", "https://github.com/user/repo.git")
	require.NoError(t, err)

	r := repo.(*Repository)
	url, err := r.getRemoteURL()
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/user/repo.git", url)
}

func TestRepository_Push_WithMockSSHAuth(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewRepository(tmpDir)
	require.NoError(t, err)

	err = repo.Init("main")
	require.NoError(t, err)

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = repo.Add(".")
	require.NoError(t, err)
	err = repo.Commit("initial commit")
	require.NoError(t, err)

	// Add a fake SSH remote (this will fail to push, but should handle auth gracefully)
	err = repo.SetRemote("origin", "git@github.com:user/repo.git")
	require.NoError(t, err)

	// Try to push - it should fail with a network/auth error, not a "no auth method" error
	err = repo.Push()
	assert.Error(t, err)
	// Should not be our "no suitable SSH authentication method found" error if SSH agent works
	// OR should be exactly that error if no SSH methods are available
	if err.Error() == "no suitable SSH authentication method found" {
		t.Skip("SSH authentication not available in test environment")
	}
}