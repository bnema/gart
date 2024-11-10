// gart/internal/system/diff_test.go

package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiffFiles(t *testing.T) {
	// Create temporary test directories that mimic real structure
	tempDir := t.TempDir()

	// Simulate ~/.config/nvim structure
	configDir := filepath.Join(tempDir, "config")
	nvimConfigDir := filepath.Join(configDir, "nvim")

	// Simulate ~/.config/gart/.store structure
	storeDir := filepath.Join(tempDir, "store")
	nvimStoreDir := filepath.Join(storeDir, "nvim")

	// Create directory structure
	dirs := []string{nvimConfigDir, nvimStoreDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory structure: %v", err)
		}
	}

	tests := []struct {
		name            string
		setupFunc       func() error
		reverseSyncMode bool
		checkFunc       func() error
		wantChanged     bool
		wantErr         bool
	}{
		{
			name: "Push mode: ~/.config/nvim/init.lua differs from store",
			setupFunc: func() error {
				// Setup different content in config and store
				if err := os.WriteFile(filepath.Join(nvimConfigDir, "init.lua"), []byte("-- Original config\nprint('hello')"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(nvimStoreDir, "init.lua"), []byte("-- Old stored config\nprint('old')"), 0644); err != nil {
					return err
				}
				return nil
			},
			reverseSyncMode: false, // Push mode
			checkFunc: func() error {
				// Verify store file was overwritten with config content
				storeContent, err := os.ReadFile(filepath.Join(nvimStoreDir, "init.lua"))
				if err != nil {
					return err
				}
				expectedContent := "-- Original config\nprint('hello')"
				if string(storeContent) != expectedContent {
					t.Errorf("Store file content = %s, want %s", string(storeContent), expectedContent)
				}
				return nil
			},
			wantChanged: true,
			wantErr:     false,
		},
		{
			name: "Pull mode: store/nvim/init.lua differs from ~/.config/nvim/init.lua",
			setupFunc: func() error {
				// Setup different content in config and store
				if err := os.WriteFile(filepath.Join(nvimStoreDir, "init.lua"), []byte("-- Store version\nprint('store')"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(nvimConfigDir, "init.lua"), []byte("-- Local version\nprint('local')"), 0644); err != nil {
					return err
				}
				return nil
			},
			reverseSyncMode: true, // Pull mode
			checkFunc: func() error {
				// Verify config file was overwritten with store content
				configContent, err := os.ReadFile(filepath.Join(nvimConfigDir, "init.lua"))
				if err != nil {
					return err
				}
				expectedContent := "-- Store version\nprint('store')"
				if string(configContent) != expectedContent {
					t.Errorf("Config file content = %s, want %s", string(configContent), expectedContent)
				}
				return nil
			},
			wantChanged: true,
			wantErr:     false,
		},
		{
			name: "Push mode: no changes needed when files are identical",
			setupFunc: func() error {
				content := "-- Same content\nprint('same')"
				if err := os.WriteFile(filepath.Join(nvimConfigDir, "init.lua"), []byte(content), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(nvimStoreDir, "init.lua"), []byte(content), 0644); err != nil {
					return err
				}
				return nil
			},
			reverseSyncMode: false, // Push mode
			checkFunc: func() error {
				// Verify files remain identical
				configContent, err := os.ReadFile(filepath.Join(nvimConfigDir, "init.lua"))
				if err != nil {
					return err
				}
				storeContent, err := os.ReadFile(filepath.Join(nvimStoreDir, "init.lua"))
				if err != nil {
					return err
				}
				if string(configContent) != string(storeContent) {
					t.Errorf("Files should be identical but differ")
				}
				return nil
			},
			wantChanged: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up test directories before each test
			cleanTestDirs := func() error {
				for _, dir := range []string{nvimConfigDir, nvimStoreDir} {
					if err := os.RemoveAll(dir); err != nil {
						return err
					}
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}
				return nil
			}

			if err := cleanTestDirs(); err != nil {
				t.Fatalf("Failed to clean test directories: %v", err)
			}

			// Setup test case
			if err := tt.setupFunc(); err != nil {
				t.Fatalf("Failed to setup test case: %v", err)
			}

			// Run the test
			changed, err := DiffFiles(nvimConfigDir, nvimStoreDir, []string{}, tt.reverseSyncMode)

			// Check results
			if (err != nil) != tt.wantErr {
				t.Errorf("DiffFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if changed != tt.wantChanged {
				t.Errorf("DiffFiles() changed = %v, want %v", changed, tt.wantChanged)
			}

			// Run additional checks
			if tt.checkFunc != nil {
				if err := tt.checkFunc(); err != nil {
					t.Errorf("Check failed: %v", err)
				}
			}
		})
	}
}
