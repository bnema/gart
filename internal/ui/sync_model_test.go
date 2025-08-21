package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnema/gart/internal/app"
	"github.com/bnema/gart/internal/config"
	"github.com/bnema/gart/internal/security"
)

func TestRunSyncView_ReverseSyncMode(t *testing.T) {
	// Create temporary test directories
	tempDir := t.TempDir()
	
	// Create source (local config) and store directories
	sourceDir := filepath.Join(tempDir, "source")
	storeDir := filepath.Join(tempDir, "store")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}

	tests := []struct {
		name            string
		reverseSyncMode bool
		setupFunc       func() error
		checkFunc       func() (bool, error)
		skipSecurity    bool
		wantResult      bool
		description     string
	}{
		{
			name:            "Push mode: sync from local config to store (directory)",
			reverseSyncMode: false,
			setupFunc: func() error {
				// Create file in source (local config directory)
				return os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("-- Local config\nlocal_setting = true"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify file was copied to store (under dotfile name subdirectory)
				storeFile := filepath.Join(storeDir, "test-dotfile", "config.lua")
				content, err := os.ReadFile(storeFile)
				if err != nil {
					return false, err
				}
				expected := "-- Local config\nlocal_setting = true"
				return string(content) == expected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should copy from local config directory to store directory",
		},
		{
			name:            "Pull mode: sync from store to local config",
			reverseSyncMode: true,
			setupFunc: func() error {
				// Create file in store first (under dotfile name subdirectory)
				storeSubDir := filepath.Join(storeDir, "test-dotfile")
				if err := os.MkdirAll(storeSubDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(storeSubDir, "config.lua"), []byte("-- Store config\nstore_setting = true"), 0644); err != nil {
					return err
				}
				// Create different content in source (local config)
				return os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("-- Old local config\nold_setting = true"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify local config was overwritten with store content
				sourceFile := filepath.Join(sourceDir, "config.lua")
				content, err := os.ReadFile(sourceFile)
				if err != nil {
					return false, err
				}
				expected := "-- Store config\nstore_setting = true"
				return string(content) == expected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should copy from store to local config directory",
		},
		{
			name:            "Pull mode: fail when store file doesn't exist",
			reverseSyncMode: true,
			setupFunc: func() error {
				// Create file only in source (local config), not in store
				return os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("-- Local only\nlocal_only = true"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify source file remains unchanged
				sourceFile := filepath.Join(sourceDir, "config.lua")
				content, err := os.ReadFile(sourceFile)
				if err != nil {
					return false, err
				}
				expected := "-- Local only\nlocal_only = true"
				return string(content) == expected, nil
			},
			skipSecurity: true,
			wantResult:   false,
			description:  "Should fail gracefully when store file doesn't exist for reverse sync",
		},
		{
			name:            "Push mode: handle directory sync",
			reverseSyncMode: false,
			setupFunc: func() error {
				// Create directory structure in source
				subDir := filepath.Join(sourceDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(subDir, "nested.lua"), []byte("-- Nested file\nnested = true"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(sourceDir, "main.lua"), []byte("-- Main file\nmain = true"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify directory structure was copied to store (under dotfile name subdirectory)
				mainFile := filepath.Join(storeDir, "test-dotfile", "main.lua")
				nestedFile := filepath.Join(storeDir, "test-dotfile", "subdir", "nested.lua")
				
				mainContent, err := os.ReadFile(mainFile)
				if err != nil {
					return false, err
				}
				nestedContent, err := os.ReadFile(nestedFile)
				if err != nil {
					return false, err
				}
				
				mainExpected := "-- Main file\nmain = true"
				nestedExpected := "-- Nested file\nnested = true"
				
				return string(mainContent) == mainExpected && string(nestedContent) == nestedExpected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should handle directory structure copying in push mode",
		},
		{
			name:            "Pull mode: handle directory sync",
			reverseSyncMode: true,
			setupFunc: func() error {
				// Create directory structure in store (under dotfile name subdirectory)
				storeMainDir := filepath.Join(storeDir, "test-dotfile")
				subDir := filepath.Join(storeMainDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(subDir, "nested.lua"), []byte("-- Store nested\nstore_nested = true"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(storeMainDir, "main.lua"), []byte("-- Store main\nstore_main = true"), 0644); err != nil {
					return err
				}
				
				// Create different content in source
				sourceSubDir := filepath.Join(sourceDir, "subdir")
				if err := os.MkdirAll(sourceSubDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(sourceSubDir, "nested.lua"), []byte("-- Local nested\nlocal_nested = true"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(sourceDir, "main.lua"), []byte("-- Local main\nlocal_main = true"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify source was overwritten with store content
				mainFile := filepath.Join(sourceDir, "main.lua")
				nestedFile := filepath.Join(sourceDir, "subdir", "nested.lua")
				
				mainContent, err := os.ReadFile(mainFile)
				if err != nil {
					return false, err
				}
				nestedContent, err := os.ReadFile(nestedFile)
				if err != nil {
					return false, err
				}
				
				mainExpected := "-- Store main\nstore_main = true"
				nestedExpected := "-- Store nested\nstore_nested = true"
				
				return string(mainContent) == mainExpected && string(nestedContent) == nestedExpected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should handle directory structure copying in pull mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean test directories
			cleanDirs := func() error {
				for _, dir := range []string{sourceDir, storeDir} {
					if err := os.RemoveAll(dir); err != nil {
						return err
					}
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}
				return nil
			}

			if err := cleanDirs(); err != nil {
				t.Fatalf("Failed to clean test directories: %v", err)
			}

			// Setup test case
			if err := tt.setupFunc(); err != nil {
				t.Fatalf("Failed to setup test case: %v", err)
			}

			// Create app instance with test configuration
			testApp := &app.App{
				Dotfile: app.Dotfile{
					Name: "test-dotfile",
					Path: sourceDir,
				},
				StoragePath: storeDir,
				Config: &config.Config{
					Settings: config.SettingsConfig{
						ReverseSyncMode: tt.reverseSyncMode,
						GitVersioning:   false, // Disable git for testing
						Security: &security.SecurityConfig{
							Enabled: false, // Disable security scanning for cleaner tests
						},
					},
				},
			}

			// Capture output to verify status messages
			result := RunSyncView(testApp, []string{}, tt.skipSecurity, nil)

			// Check result matches expectation
			if result != tt.wantResult {
				t.Errorf("RunSyncView() result = %v, want %v", result, tt.wantResult)
			}

			// Run additional verification
			if tt.checkFunc != nil {
				passed, err := tt.checkFunc()
				if err != nil {
					t.Errorf("Check function failed: %v", err)
				}
				if !passed {
					t.Errorf("Check function validation failed: %s", tt.description)
				}
			}
		})
	}
}

func TestRunSyncView_StatusMessages(t *testing.T) {
	// This test captures output to verify correct status messages
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	storeDir := filepath.Join(tempDir, "store")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}

	tests := []struct {
		name            string
		reverseSyncMode bool
		setupFunc       func() error
		expectedMessage string
	}{
		{
			name:            "Push mode status message",
			reverseSyncMode: false,
			setupFunc: func() error {
				return os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("content"), 0644)
			},
			expectedMessage: "Updating store",
		},
		{
			name:            "Pull mode status message",
			reverseSyncMode: true,
			setupFunc: func() error {
				// Create store subdirectory and file
				storeSubDir := filepath.Join(storeDir, "test-dotfile")
				if err := os.MkdirAll(storeSubDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(storeSubDir, "config.lua"), []byte("store content"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("local content"), 0644)
			},
			expectedMessage: "Updating local config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean directories
			for _, dir := range []string{sourceDir, storeDir} {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatalf("Failed to remove directory: %v", err)
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			// Setup test case
			if err := tt.setupFunc(); err != nil {
				t.Fatalf("Failed to setup test case: %v", err)
			}

			// Create app instance
			testApp := &app.App{
				Dotfile: app.Dotfile{
					Name: "test-dotfile",
					Path: sourceDir,
				},
				StoragePath: storeDir,
				Config: &config.Config{
					Settings: config.SettingsConfig{
						ReverseSyncMode: tt.reverseSyncMode,
						GitVersioning:   false,
						Security: &security.SecurityConfig{
							Enabled: false,
						},
					},
				},
			}

			// Note: This is a simplified test - in a real scenario, you'd want to capture
			// the actual output and verify it contains the expected message.
			// For now, we just verify the function runs without error
			result := RunSyncView(testApp, []string{}, true, nil)
			if !result {
				t.Errorf("RunSyncView() should have succeeded but returned false")
			}
		})
	}
}

func TestRunSyncView_NoChanges(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	storeDir := filepath.Join(tempDir, "store")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}

	tests := []struct {
		name            string
		reverseSyncMode bool
		setupFunc       func() error
	}{
		{
			name:            "Push mode: no changes when files identical",
			reverseSyncMode: false,
			setupFunc: func() error {
				content := "-- Same content\nsame = true"
				if err := os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte(content), 0644); err != nil {
					return err
				}
				// Create store subdirectory and file
				storeSubDir := filepath.Join(storeDir, "test-dotfile")
				if err := os.MkdirAll(storeSubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(storeSubDir, "config.lua"), []byte(content), 0644)
			},
		},
		{
			name:            "Pull mode: no changes when files identical",
			reverseSyncMode: true,
			setupFunc: func() error {
				content := "-- Same content\nsame = true"
				if err := os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte(content), 0644); err != nil {
					return err
				}
				// Create store subdirectory and file
				storeSubDir := filepath.Join(storeDir, "test-dotfile")
				if err := os.MkdirAll(storeSubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(storeSubDir, "config.lua"), []byte(content), 0644)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean directories
			for _, dir := range []string{sourceDir, storeDir} {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatalf("Failed to remove directory: %v", err)
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			// Setup test case
			if err := tt.setupFunc(); err != nil {
				t.Fatalf("Failed to setup test case: %v", err)
			}

			// Create app instance
			testApp := &app.App{
				Dotfile: app.Dotfile{
					Name: "test-dotfile",
					Path: sourceDir,
				},
				StoragePath: storeDir,
				Config: &config.Config{
					Settings: config.SettingsConfig{
						ReverseSyncMode: tt.reverseSyncMode,
						GitVersioning:   false,
						Security: &security.SecurityConfig{
							Enabled: false,
						},
					},
				},
			}

			// Run sync - should return true but not make any changes
			result := RunSyncView(testApp, []string{}, true, nil)
			if !result {
				t.Errorf("RunSyncView() should have succeeded but returned false")
			}

			// Verify files remain unchanged
			sourceContent, err := os.ReadFile(filepath.Join(sourceDir, "config.lua"))
			if err != nil {
				t.Fatalf("Failed to read source file: %v", err)
			}
			storeContent, err := os.ReadFile(filepath.Join(storeDir, "test-dotfile", "config.lua"))
			if err != nil {
				t.Fatalf("Failed to read store file: %v", err)
			}

			if !strings.Contains(string(sourceContent), "same = true") {
				t.Errorf("Source file should remain unchanged")
			}
			if !strings.Contains(string(storeContent), "same = true") {
				t.Errorf("Store file should remain unchanged") 
			}
		})
	}
}

func TestRunSyncView_SingleFileReverseSyncMode(t *testing.T) {
	// Create temporary test directories
	tempDir := t.TempDir()
	
	// For single files, we need the actual file paths
	sourceFile := filepath.Join(tempDir, "source", "config.conf")
	storeDir := filepath.Join(tempDir, "store")
	
	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}

	tests := []struct {
		name            string
		reverseSyncMode bool
		setupFunc       func() error
		checkFunc       func() (bool, error)
		skipSecurity    bool
		wantResult      bool
		description     string
	}{
		{
			name:            "Push mode: sync single file from local config to store",
			reverseSyncMode: false,
			setupFunc: func() error {
				return os.WriteFile(sourceFile, []byte("# Single file config\nsetting = value"), 0644)
			},
			checkFunc: func() (bool, error) {
				// For single files, store path is dotfile-name + extension directly in storage
				storeFile := filepath.Join(storeDir, "test-dotfile.conf")
				content, err := os.ReadFile(storeFile)
				if err != nil {
					return false, err
				}
				expected := "# Single file config\nsetting = value"
				return string(content) == expected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should copy single file from local config to store",
		},
		{
			name:            "Pull mode: sync single file from store to local config",
			reverseSyncMode: true,
			setupFunc: func() error {
				// Create single file in store with dotfile-name.ext format
				if err := os.WriteFile(filepath.Join(storeDir, "test-dotfile.conf"), []byte("# Store file config\nstore_setting = store_value"), 0644); err != nil {
					return err
				}
				// Create different content in source file
				return os.WriteFile(sourceFile, []byte("# Local file config\nlocal_setting = local_value"), 0644)
			},
			checkFunc: func() (bool, error) {
				// Verify source file was overwritten with store content
				content, err := os.ReadFile(sourceFile)
				if err != nil {
					return false, err
				}
				expected := "# Store file config\nstore_setting = store_value"
				return string(content) == expected, nil
			},
			skipSecurity: true,
			wantResult:   true,
			description:  "Should copy single file from store to local config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean test files
			cleanFiles := func() error {
				// Remove source file
				if err := os.Remove(sourceFile); err != nil && !os.IsNotExist(err) {
					return err
				}
				// Remove store files
				if err := os.RemoveAll(storeDir); err != nil {
					return err
				}
				if err := os.MkdirAll(storeDir, 0755); err != nil {
					return err
				}
				return nil
			}

			if err := cleanFiles(); err != nil {
				t.Fatalf("Failed to clean test files: %v", err)
			}

			// Setup test case
			if err := tt.setupFunc(); err != nil {
				t.Fatalf("Failed to setup test case: %v", err)
			}

			// Create app instance with test configuration for single file
			testApp := &app.App{
				Dotfile: app.Dotfile{
					Name: "test-dotfile",
					Path: sourceFile, // Point to the actual file, not directory
				},
				StoragePath: storeDir,
				Config: &config.Config{
					Settings: config.SettingsConfig{
						ReverseSyncMode: tt.reverseSyncMode,
						GitVersioning:   false, // Disable git for testing
						Security: &security.SecurityConfig{
							Enabled: false, // Disable security scanning for cleaner tests
						},
					},
				},
			}

			// Run sync
			result := RunSyncView(testApp, []string{}, tt.skipSecurity, nil)

			// Check result matches expectation
			if result != tt.wantResult {
				t.Errorf("RunSyncView() result = %v, want %v", result, tt.wantResult)
			}

			// Run additional verification
			if tt.checkFunc != nil {
				passed, err := tt.checkFunc()
				if err != nil {
					t.Errorf("Check function failed: %v", err)
				}
				if !passed {
					t.Errorf("Check function validation failed: %s", tt.description)
				}
			}
		})
	}
}

func TestRunSyncView_SkipAllFunctionality(t *testing.T) {
	// Test that skipAll flag properly propagates across multiple dotfile syncs
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	storeDir := filepath.Join(tempDir, "store")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}
	
	// Create test file
	if err := os.WriteFile(filepath.Join(sourceDir, "config.lua"), []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create test app with security enabled
	testApp := &app.App{
		ConfigFilePath: filepath.Join(tempDir, "config.toml"),
		StoragePath:    storeDir,
		Dotfile: app.Dotfile{
			Name: "test-dotfile",
			Path: sourceDir,
		},
		Config: &config.Config{
			Settings: config.SettingsConfig{
				StoragePath:     storeDir,
				GitVersioning:   false,
				ReverseSyncMode: false,
				Security: &security.SecurityConfig{
					Enabled:     true,
					Interactive: true,
				},
			},
			Dotfiles:        map[string]string{"test-dotfile": sourceDir},
			DotfilesIgnores: map[string][]string{},
		},
	}
	
	tests := []struct {
		name           string
		initialSkipAll bool
		expectedSkipAll bool
		description    string
	}{
		{
			name:            "Initial skipAll false, should remain false",
			initialSkipAll:  false,
			expectedSkipAll: false,
			description:     "When no security issues are found, skipAll should remain false",
		},
		{
			name:            "Initial skipAll true, should remain true",
			initialSkipAll:  true,
			expectedSkipAll: true,
			description:     "When skipAll is already set, should bypass security entirely",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean test directories
			if err := os.RemoveAll(storeDir); err != nil {
				t.Fatalf("Failed to clean store directory: %v", err)
			}
			if err := os.MkdirAll(storeDir, 0755); err != nil {
				t.Fatalf("Failed to recreate store directory: %v", err)
			}
			
			// Set up initial skipAll flag
			skipAllFlag := tt.initialSkipAll
			
			// Run sync with skipAll pointer
			result := RunSyncView(testApp, []string{}, false, &skipAllFlag)
			
			// Should succeed
			if !result {
				t.Errorf("RunSyncView() should have succeeded but returned false")
			}
			
			// Check that skipAll flag matches expectation
			if skipAllFlag != tt.expectedSkipAll {
				t.Errorf("skipAll flag = %v, want %v", skipAllFlag, tt.expectedSkipAll)
			}
		})
	}
}

func TestRunSyncView_SecurityMessages(t *testing.T) {
	// Test that security messages are only shown when security is enabled
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	storeDir := filepath.Join(tempDir, "store")
	
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("Failed to create store directory: %v", err)
	}

	tests := []struct {
		name             string
		securityEnabled  bool
		skipSecurity     bool
		shouldShowMessages bool
		description      string
	}{
		{
			name:             "Security enabled in config, no skip flag",
			securityEnabled:  true,
			skipSecurity:     false,
			shouldShowMessages: true,
			description:      "Should show security messages when security is enabled and not skipped",
		},
		{
			name:             "Security disabled in config, no skip flag",
			securityEnabled:  false,
			skipSecurity:     false,
			shouldShowMessages: false,
			description:      "Should NOT show security messages when security is disabled in config",
		},
		{
			name:             "Security enabled in config, but skip flag used",
			securityEnabled:  true,
			skipSecurity:     true,
			shouldShowMessages: false,
			description:      "Should NOT show security messages when skip flag is used",
		},
		{
			name:             "Security disabled in config and skip flag used",
			securityEnabled:  false,
			skipSecurity:     true,
			shouldShowMessages: false,
			description:      "Should NOT show security messages when both config disabled and skip flag used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean directories
			for _, dir := range []string{sourceDir, storeDir} {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatalf("Failed to remove directory: %v", err)
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			// Create a file that will cause changes (to trigger sync)
			if err := os.WriteFile(filepath.Join(sourceDir, "test.conf"), []byte("test content"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create app instance with specific security configuration
			testApp := &app.App{
				Dotfile: app.Dotfile{
					Name: "test-dotfile",
					Path: sourceDir,
				},
				StoragePath: storeDir,
				Config: &config.Config{
					Settings: config.SettingsConfig{
						ReverseSyncMode: false,
						GitVersioning:   false,
						Security: &security.SecurityConfig{
							Enabled: tt.securityEnabled,
						},
					},
				},
			}

			// For this test, we would ideally capture stdout/stderr to verify
			// the presence or absence of security messages.
			// For now, we verify that the function behavior is correct
			// Pass nil for skipAllSecurity since this is testing single operations
			result := RunSyncView(testApp, []string{}, tt.skipSecurity, nil)
			
			// The function should always succeed in this test case
			if !result {
				t.Errorf("RunSyncView() should have succeeded but returned false")
			}

			// Verify the sync actually happened (file was copied to store)
			expectedStoreFile := filepath.Join(storeDir, "test-dotfile", "test.conf")
			if _, err := os.Stat(expectedStoreFile); os.IsNotExist(err) {
				t.Errorf("Store file should have been created but doesn't exist: %s", expectedStoreFile)
			}

			// Note: In a more comprehensive test, we would capture the actual output
			// and verify that security messages are present or absent based on
			// tt.shouldShowMessages. This could be done by redirecting stdout/stderr
			// or by modifying the function to return additional information about
			// what messages were displayed.
		})
	}
}