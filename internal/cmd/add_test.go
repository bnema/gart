package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/system"
	"github.com/stretchr/testify/assert"
)

func setupTestEnvironment(t *testing.T) (string, string, func()) {
	// Create temporary directories for test
	tmpRoot, err := os.MkdirTemp("", "gart-test-*")
	assert.NoError(t, err)

	// Create temp config and storage directories
	configDir := filepath.Join(tmpRoot, "config")
	storageDir := filepath.Join(tmpRoot, "storage")

	err = os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(storageDir, 0755)
	assert.NoError(t, err)

	// Create test dotfile directory
	testDotfileDir := filepath.Join(tmpRoot, "dotfiles")
	err = os.MkdirAll(testDotfileDir, 0755)
	assert.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tmpRoot)
	}

	return testDotfileDir, storageDir, cleanup
}

func createTestFiles(t *testing.T, baseDir string, files []string, dirs []string) {
	// Create directories
	for _, dir := range dirs {
		dirPath := filepath.Join(baseDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		assert.NoError(t, err)
	}

	// Create files
	for _, file := range files {
		filePath := filepath.Join(baseDir, file)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filePath, []byte("test content"), 0644)
		assert.NoError(t, err)
	}
}

func TestAddCommand(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		dirs           []string
		ignorePatterns []string
		expectedFiles  []string
		expectedDirs   []string
	}{
		{
			name: "Basic file ignore",
			files: []string{
				"config.json",
				"settings.yaml",
				"ignore.json",
			},
			ignorePatterns: []string{"*.json"},
			expectedFiles:  []string{"settings.yaml"},
		},
		{
			name: "Multiple chained ignore flags",
			files: []string{
				"main.swp",
				"config.bak",
				"node_modules/package.json",
				"src/index.js",
			},
			ignorePatterns: []string{"*.swp", "*.bak", "node_modules/"},
			expectedFiles:  []string{"src/index.js"},
		},
		{
			name: "Directory ignore with trailing slash",
			dirs: []string{
				"cache",
				"temp",
				"data",
			},
			files: []string{
				"cache/file1.txt",
				"temp/file2.txt",
				"data/file3.txt",
			},
			ignorePatterns: []string{"cache/", "temp/"},
			expectedDirs:   []string{"data"},
			expectedFiles:  []string{"data/file3.txt"},
		},
		{
			name: "Directory ignore without trailing slash",
			dirs: []string{
				"node_modules",
				"dist",
				"src",
			},
			files: []string{
				"node_modules/package.json",
				"dist/bundle.js",
				"src/main.js",
			},
			ignorePatterns: []string{"node_modules", "dist"},
			expectedDirs:   []string{"src"},
			expectedFiles:  []string{"src/main.js"},
		},
		{
			name: "Nested wildcard patterns",
			dirs: []string{
				"src/cache",
				"build/cache",
				"test/cache",
				"src/data",
			},
			files: []string{
				"src/cache/temp.log",
				"build/cache/build.log",
				"test/cache/test.log",
				"src/data/config.json",
			},
			ignorePatterns: []string{"**/cache/", "*.log"},
			expectedDirs:   []string{"src", "src/data"},
			expectedFiles:  []string{"src/data/config.json"},
		},
		{
			name: "Complex pattern combination",
			dirs: []string{
				"src/tests",
				"src/cache",
				"dist",
			},
			files: []string{
				"src/tests/test.spec.js",
				"src/cache/temp.dat",
				"dist/bundle.js",
				"README.md",
				"package.json",
			},
			ignorePatterns: []string{
				"**/cache/",
				"dist/",
				"*.spec.js",
				"package.json",
			},
			expectedDirs: []string{
				"src",
				"src/tests",
			},
			expectedFiles: []string{
				"README.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			dotfileDir, storageDir, cleanup := setupTestEnvironment(t)
			defer cleanup()

			// Create test files and directories
			createTestFiles(t, dotfileDir, tt.files, tt.dirs)

			// Create config file
			configPath := filepath.Join(filepath.Dir(storageDir), "config", "config.toml")
			err := os.MkdirAll(filepath.Dir(configPath), 0755)
			assert.NoError(t, err)

			// Create and initialize config
			cfg := &config.Config{
				Settings: config.SettingsConfig{
					StoragePath: storageDir,
				},
				Dotfiles: make(map[string]string),
			}
			err = config.SaveConfig(configPath, cfg)
			assert.NoError(t, err)

			// Initialize app instance
			app := &app.App{
				ConfigFilePath: configPath,
				StoragePath:    storageDir,
				Config:         cfg,
				Dotfile: app.Dotfile{ // Changed from pointer to value type
					Name: "testdotfile",
					Path: dotfileDir,
				},
			}

			// Run add command
			err = app.AddDotfile(dotfileDir, "testdotfile", tt.ignorePatterns)
			assert.NoError(t, err)

			// Verify results
			for _, expectedFile := range tt.expectedFiles {
				storagePath := filepath.Join(storageDir, "testdotfile", expectedFile)
				_, err := os.Stat(storagePath)
				assert.NoError(t, err, "Expected file not found: %s", expectedFile)
			}

			for _, expectedDir := range tt.expectedDirs {
				storagePath := filepath.Join(storageDir, "testdotfile", expectedDir)
				_, err := os.Stat(storagePath)
				assert.NoError(t, err, "Expected directory not found: %s", expectedDir)
			}

			// Verify ignored files/dirs don't exist
			for _, file := range tt.files {
				if !contains(tt.expectedFiles, file) {
					storagePath := filepath.Join(storageDir, "testdotfile", file)
					_, err := os.Stat(storagePath)
					assert.True(t, os.IsNotExist(err), "Ignored file should not exist: %s", file)
				}
			}

			for _, dir := range tt.dirs {
				if !contains(tt.expectedDirs, dir) {
					storagePath := filepath.Join(storageDir, "testdotfile", dir)
					_, err := os.Stat(storagePath)
					assert.True(t, os.IsNotExist(err), "Ignored directory should not exist: %s", dir)
				}
			}
		})
	}
}

func TestCopyDirectory(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "copy-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create source directory structure
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	assert.NoError(t, err)

	// Create test directory and file
	testDir := filepath.Join(srcDir, "test")
	err = os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)

	testFile := filepath.Join(testDir, "something.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Create destination directory
	dstDir := filepath.Join(tmpDir, "dst")

	// Test copying with ignore pattern
	ignores := []string{"test/"}
	err = system.CopyDirectory(srcDir, dstDir, ignores)
	assert.NoError(t, err)

	// Verify that the test directory was not created in destination
	ignoredDir := filepath.Join(dstDir, "test")
	_, err = os.Stat(ignoredDir)
	assert.True(t, os.IsNotExist(err), "Directory 'test' should not exist in destination")

	// Verify that the file inside ignored directory was not copied
	ignoredFile := filepath.Join(dstDir, "test", "something.txt")
	_, err = os.Stat(ignoredFile)
	assert.True(t, os.IsNotExist(err), "File 'something.txt' should not exist in destination")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
