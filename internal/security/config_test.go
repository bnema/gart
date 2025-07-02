package security

import (
	"testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	// Test default values
	if !config.Enabled {
		t.Error("DefaultSecurityConfig() should enable security by default")
	}

	if !config.ScanContent {
		t.Error("DefaultSecurityConfig() should enable content scanning by default")
	}

	if !config.ExcludePatterns {
		t.Error("DefaultSecurityConfig() should enable pattern exclusion by default")
	}

	if config.Sensitivity != SensitivityMedium {
		t.Errorf("DefaultSecurityConfig() sensitivity = %v, want %v", config.Sensitivity, SensitivityMedium)
	}

	if !config.FailOnSecrets {
		t.Error("DefaultSecurityConfig() should fail on secrets by default")
	}

	if !config.Interactive {
		t.Error("DefaultSecurityConfig() should be interactive by default")
	}

	// Test content scan defaults
	if config.ContentScan.EntropyThreshold != MediumEntropyThreshold {
		t.Errorf("DefaultSecurityConfig() entropy threshold = %f, want %f",
			config.ContentScan.EntropyThreshold, MediumEntropyThreshold)
	}

	if config.ContentScan.MinSecretLength != 20 {
		t.Errorf("DefaultSecurityConfig() min secret length = %d, want 20", config.ContentScan.MinSecretLength)
	}

	if config.ContentScan.MaxFileSize != 10*1024*1024 {
		t.Errorf("DefaultSecurityConfig() max file size = %d, want %d",
			config.ContentScan.MaxFileSize, 10*1024*1024)
	}

	if config.ContentScan.ScanBinaryFiles {
		t.Error("DefaultSecurityConfig() should not scan binary files by default")
	}

	// Test allowlist defaults
	expectedPatterns := []string{"EXAMPLE_*", "DEMO_*", "TEST_*"}
	if len(config.Allowlist.Patterns) != len(expectedPatterns) {
		t.Errorf("DefaultSecurityConfig() allowlist patterns count = %d, want %d",
			len(config.Allowlist.Patterns), len(expectedPatterns))
	}

	for i, expected := range expectedPatterns {
		if i >= len(config.Allowlist.Patterns) || config.Allowlist.Patterns[i] != expected {
			t.Errorf("DefaultSecurityConfig() allowlist pattern[%d] = %q, want %q",
				i, config.Allowlist.Patterns[i], expected)
		}
	}
}

func TestSecurityConfig_Merge(t *testing.T) {
	base := DefaultSecurityConfig()
	base.Enabled = true
	base.Sensitivity = SensitivityMedium
	base.PatternConfig.Custom = []string{"base_pattern"}
	base.Allowlist.Patterns = []string{"BASE_*"}

	other := &SecurityConfig{
		Enabled:     false,
		Sensitivity: SensitivityHigh,
		PatternConfig: PatternConfig{
			Critical: []string{"*.critical"},
			Custom:   []string{"other_pattern"},
		},
		ContentScan: ContentScanConfig{
			EntropyThreshold: 5.0,
			MinSecretLength:  30,
		},
		Allowlist: AllowlistConfig{
			Patterns: []string{"OTHER_*"},
			Files:    []string{"allowed.txt"},
		},
	}

	base.Merge(other)

	// Test that other's values override base's values
	if base.Enabled != false {
		t.Errorf("Merge() enabled = %v, want false", base.Enabled)
	}

	if base.Sensitivity != SensitivityHigh {
		t.Errorf("Merge() sensitivity = %v, want %v", base.Sensitivity, SensitivityHigh)
	}

	if base.ContentScan.EntropyThreshold != 5.0 {
		t.Errorf("Merge() entropy threshold = %f, want 5.0", base.ContentScan.EntropyThreshold)
	}

	if base.ContentScan.MinSecretLength != 30 {
		t.Errorf("Merge() min secret length = %d, want 30", base.ContentScan.MinSecretLength)
	}

	// Test that arrays are appended
	expectedCritical := []string{"*.critical"}
	if len(base.PatternConfig.Critical) != len(expectedCritical) {
		t.Errorf("Merge() critical patterns count = %d, want %d",
			len(base.PatternConfig.Critical), len(expectedCritical))
	}

	expectedCustom := []string{"base_pattern", "other_pattern"}
	if len(base.PatternConfig.Custom) != len(expectedCustom) {
		t.Errorf("Merge() custom patterns count = %d, want %d",
			len(base.PatternConfig.Custom), len(expectedCustom))
	}

	expectedAllowlistPatterns := []string{"BASE_*", "OTHER_*"}
	if len(base.Allowlist.Patterns) != len(expectedAllowlistPatterns) {
		t.Errorf("Merge() allowlist patterns count = %d, want %d",
			len(base.Allowlist.Patterns), len(expectedAllowlistPatterns))
	}

	expectedAllowlistFiles := []string{"allowed.txt"}
	if len(base.Allowlist.Files) != len(expectedAllowlistFiles) {
		t.Errorf("Merge() allowlist files count = %d, want %d",
			len(base.Allowlist.Files), len(expectedAllowlistFiles))
	}
}

func TestSecurityConfig_MergeNil(t *testing.T) {
	base := DefaultSecurityConfig()
	originalEnabled := base.Enabled
	originalSensitivity := base.Sensitivity

	base.Merge(nil)

	// Should remain unchanged
	if base.Enabled != originalEnabled {
		t.Errorf("Merge(nil) changed enabled from %v to %v", originalEnabled, base.Enabled)
	}

	if base.Sensitivity != originalSensitivity {
		t.Errorf("Merge(nil) changed sensitivity from %v to %v", originalSensitivity, base.Sensitivity)
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *SecurityConfig
		expectError bool
		errorField  string
	}{
		{
			name:        "valid default config",
			config:      DefaultSecurityConfig(),
			expectError: false,
		},
		{
			name: "invalid entropy threshold - too low",
			config: &SecurityConfig{
				Sensitivity: SensitivityMedium,
				ContentScan: ContentScanConfig{
					EntropyThreshold: -1.0,
				},
			},
			expectError: true,
			errorField:  "entropy_threshold",
		},
		{
			name: "invalid entropy threshold - too high",
			config: &SecurityConfig{
				Sensitivity: SensitivityMedium,
				ContentScan: ContentScanConfig{
					EntropyThreshold: 10.0,
				},
			},
			expectError: true,
			errorField:  "entropy_threshold",
		},
		{
			name: "invalid min secret length",
			config: &SecurityConfig{
				Sensitivity: SensitivityMedium,
				ContentScan: ContentScanConfig{
					MinSecretLength: 0,
				},
			},
			expectError: true,
			errorField:  "min_secret_length",
		},
		{
			name: "invalid max file size",
			config: &SecurityConfig{
				Sensitivity: SensitivityMedium,
				ContentScan: ContentScanConfig{
					MaxFileSize: -1,
				},
			},
			expectError: true,
			errorField:  "max_file_size",
		},
		{
			name: "invalid context window",
			config: &SecurityConfig{
				Sensitivity: SensitivityMedium,
				ContentScan: ContentScanConfig{
					ContextWindow: -1,
				},
			},
			expectError: true,
			errorField:  "context_window",
		},
		{
			name: "invalid sensitivity",
			config: &SecurityConfig{
				Sensitivity: "invalid",
			},
			expectError: true,
			errorField:  "sensitivity",
		},
		{
			name: "valid edge case values",
			config: &SecurityConfig{
				Sensitivity: SensitivityParanoid,
				ContentScan: ContentScanConfig{
					EntropyThreshold: 0.0,
					MinSecretLength:  1,
					MaxFileSize:      0,
					ContextWindow:    0,
					ScanBinaryFiles:  true,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error for field %s, got nil", tt.errorField)
				} else {
					validationErr, ok := err.(*ValidationError)
					if !ok {
						t.Errorf("Validate() expected ValidationError, got %T", err)
					} else if validationErr.Field != tt.errorField {
						t.Errorf("Validate() error field = %s, want %s", validationErr.Field, tt.errorField)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSecurityConfig_IsFileAllowed(t *testing.T) {
	config := &SecurityConfig{
		Allowlist: AllowlistConfig{
			Files: []string{".gitconfig", "safe.conf", "/path/to/allowed.txt"},
		},
	}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "allowed file - exact match",
			filePath: ".gitconfig",
			expected: true,
		},
		{
			name:     "allowed file - another exact match",
			filePath: "safe.conf",
			expected: true,
		},
		{
			name:     "allowed file - full path",
			filePath: "/path/to/allowed.txt",
			expected: true,
		},
		{
			name:     "not allowed file",
			filePath: ".env",
			expected: false,
		},
		{
			name:     "partial match should not be allowed",
			filePath: "prefix.gitconfig",
			expected: false,
		},
		{
			name:     "empty path",
			filePath: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsFileAllowed(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsFileAllowed(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestSecurityConfig_IsPatternAllowed(t *testing.T) {
	config := &SecurityConfig{
		Allowlist: AllowlistConfig{
			Patterns: []string{"EXAMPLE_*", "*_TEST", "DEMO_API_KEY", "*"},
		},
	}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "prefix pattern match",
			value:    "EXAMPLE_SECRET_KEY",
			expected: true,
		},
		{
			name:     "suffix pattern match",
			value:    "MY_SECRET_TEST",
			expected: true,
		},
		{
			name:     "exact pattern match",
			value:    "DEMO_API_KEY",
			expected: true,
		},
		{
			name:     "wildcard match",
			value:    "anything_at_all",
			expected: true,
		},
		{
			name:     "no match",
			value:    "SECRET_KEY",
			expected: false,
		},
	}

	// Test without wildcard
	configNoWildcard := &SecurityConfig{
		Allowlist: AllowlistConfig{
			Patterns: []string{"EXAMPLE_*", "*_TEST", "DEMO_API_KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsPatternAllowed(tt.value)
			if tt.name == "wildcard match" {
				if !result {
					t.Errorf("IsPatternAllowed(%q) with wildcard = %v, want true", tt.value, result)
				}
			} else {
				// Test with the non-wildcard config for other cases
				result = configNoWildcard.IsPatternAllowed(tt.value)
				expected := tt.expected && tt.name != "wildcard match"
				if result != expected {
					t.Errorf("IsPatternAllowed(%q) = %v, want %v", tt.value, result, expected)
				}
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		pattern  string
		expected bool
	}{
		{
			name:     "wildcard pattern",
			value:    "anything",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "prefix pattern",
			value:    "EXAMPLE_SECRET",
			pattern:  "EXAMPLE_*",
			expected: true,
		},
		{
			name:     "suffix pattern",
			value:    "SECRET_TEST",
			pattern:  "*_TEST",
			expected: true,
		},
		{
			name:     "exact match",
			value:    "SECRET_KEY",
			pattern:  "SECRET_KEY",
			expected: true,
		},
		{
			name:     "prefix no match",
			value:    "OTHER_SECRET",
			pattern:  "EXAMPLE_*",
			expected: false,
		},
		{
			name:     "suffix no match",
			value:    "SECRET_PROD",
			pattern:  "*_TEST",
			expected: false,
		},
		{
			name:     "exact no match",
			value:    "SECRET_KEY",
			pattern:  "API_KEY",
			expected: false,
		},
		{
			name:     "empty value",
			value:    "",
			pattern:  "EXAMPLE_*",
			expected: false,
		},
		{
			name:     "empty pattern",
			value:    "SECRET",
			pattern:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.value, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v",
					tt.value, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "validation error in field 'test_field': test message"
	result := err.Error()

	if result != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", result, expected)
	}
}
