package security

import (
	"testing"
)

func TestPatternMatcher_ShouldExclude(t *testing.T) {
	config := DefaultSecurityConfig()
	pm := NewPatternMatcher(config)

	tests := []struct {
		name          string
		filePath      string
		shouldExclude bool
		expectedRisk  RiskLevel
	}{
		// Critical patterns
		{
			name:          "env file",
			filePath:      "/home/user/.env",
			shouldExclude: true,
			expectedRisk:  RiskLevelCritical,
		},
		{
			name:          "env with extension",
			filePath:      "/home/user/.env.local",
			shouldExclude: true,
			expectedRisk:  RiskLevelCritical,
		},
		{
			name:          "private key",
			filePath:      "/home/user/.ssh/id_rsa",
			shouldExclude: true,
			expectedRisk:  RiskLevelCritical,
		},
		{
			name:          "AWS credentials",
			filePath:      "/home/user/.aws/credentials",
			shouldExclude: true,
			expectedRisk:  RiskLevelCritical,
		},
		{
			name:          "docker config",
			filePath:      "/home/user/.docker/config.json",
			shouldExclude: true,
			expectedRisk:  RiskLevelCritical,
		},

		// High risk patterns
		{
			name:          "bash history",
			filePath:      "/home/user/.bash_history",
			shouldExclude: true,
			expectedRisk:  RiskLevelHigh,
		},
		{
			name:          "git credentials",
			filePath:      "/home/user/.git-credentials",
			shouldExclude: true,
			expectedRisk:  RiskLevelHigh,
		},
		{
			name:          "npm config",
			filePath:      "/home/user/.npmrc",
			shouldExclude: true,
			expectedRisk:  RiskLevelHigh,
		},

		// Medium risk patterns
		{
			name:          "git config",
			filePath:      "/home/user/.gitconfig",
			shouldExclude: true,
			expectedRisk:  RiskLevelMedium,
		},
		{
			name:          "log file",
			filePath:      "/var/log/app.log",
			shouldExclude: true,
			expectedRisk:  RiskLevelMedium,
		},

		// Low risk patterns
		{
			name:          "DS_Store",
			filePath:      "/home/user/.DS_Store",
			shouldExclude: true,
			expectedRisk:  RiskLevelLow,
		},

		// Safe files
		{
			name:          "regular config file",
			filePath:      "/home/user/.vimrc",
			shouldExclude: false,
			expectedRisk:  RiskLevelNone,
		},
		{
			name:          "shell config",
			filePath:      "/home/user/.bashrc",
			shouldExclude: false,
			expectedRisk:  RiskLevelNone,
		},
		{
			name:          "regular text file",
			filePath:      "/home/user/document.txt",
			shouldExclude: false,
			expectedRisk:  RiskLevelNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldExclude, risk, _ := pm.ShouldExclude(tt.filePath)

			if shouldExclude != tt.shouldExclude {
				t.Errorf("ShouldExclude(%q) exclude = %v, want %v", tt.filePath, shouldExclude, tt.shouldExclude)
			}

			if tt.shouldExclude && risk != tt.expectedRisk {
				t.Errorf("ShouldExclude(%q) risk = %v, want %v", tt.filePath, risk, tt.expectedRisk)
			}
		})
	}
}

func TestPatternMatcher_MatchPattern(t *testing.T) {
	pm := NewPatternMatcher(nil)

	tests := []struct {
		name     string
		fullPath string
		baseName string
		pattern  string
		expected bool
	}{
		{
			name:     "exact basename match",
			fullPath: "/home/user/.env",
			baseName: ".env",
			pattern:  ".env",
			expected: true,
		},
		{
			name:     "glob pattern match",
			fullPath: "/home/user/.env.local",
			baseName: ".env.local",
			pattern:  ".env.*",
			expected: true,
		},
		{
			name:     "directory pattern",
			fullPath: "/home/user/.ssh/id_rsa",
			baseName: "id_rsa",
			pattern:  ".ssh/*",
			expected: true,
		},
		{
			name:     "wildcard pattern",
			fullPath: "/home/user/secret.key",
			baseName: "secret.key",
			pattern:  "*.key",
			expected: true,
		},
		{
			name:     "no match",
			fullPath: "/home/user/.vimrc",
			baseName: ".vimrc",
			pattern:  ".env",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.matchPattern(tt.fullPath, tt.baseName, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q, %q) = %v, want %v",
					tt.fullPath, tt.baseName, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestPatternMatcher_AddRemovePattern(t *testing.T) {
	pm := NewPatternMatcher(nil)

	// Test adding pattern
	customPattern := "*.secret"
	pm.AddPattern(customPattern, RiskLevelHigh)

	patterns := pm.GetPatterns(RiskLevelHigh)
	found := false
	for _, pattern := range patterns {
		if pattern == customPattern {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("AddPattern failed: pattern %q not found in high risk patterns", customPattern)
	}

	// Test pattern matching
	shouldExclude, risk, _ := pm.ShouldExclude("/home/user/api.secret")
	if !shouldExclude || risk != RiskLevelHigh {
		t.Errorf("Custom pattern not working: shouldExclude=%v, risk=%v", shouldExclude, risk)
	}

	// Test removing pattern
	pm.RemovePattern(customPattern, RiskLevelHigh)

	patterns = pm.GetPatterns(RiskLevelHigh)
	found = false
	for _, pattern := range patterns {
		if pattern == customPattern {
			found = true
			break
		}
	}

	if found {
		t.Errorf("RemovePattern failed: pattern %q still found in high risk patterns", customPattern)
	}
}

func TestPatternMatcher_GetAllPatterns(t *testing.T) {
	config := DefaultSecurityConfig()
	pm := NewPatternMatcher(config)

	allPatterns := pm.GetAllPatterns()

	// Check that we have patterns for each risk level
	expectedLevels := []RiskLevel{RiskLevelCritical, RiskLevelHigh, RiskLevelMedium, RiskLevelLow}

	for _, level := range expectedLevels {
		patterns, exists := allPatterns[level]
		if !exists || len(patterns) == 0 {
			t.Errorf("No patterns found for risk level %v", level)
		}
	}

	// Check for some expected critical patterns
	criticalPatterns := allPatterns[RiskLevelCritical]
	expectedCritical := []string{".env", "*.key", ".ssh/id_*"}

	for _, expected := range expectedCritical {
		found := false
		for _, pattern := range criticalPatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected critical pattern %q not found", expected)
		}
	}
}

func TestPatternMatcher_CustomConfig(t *testing.T) {
	config := &SecurityConfig{
		PatternConfig: PatternConfig{
			Critical: []string{"*.custom"},
			High:     []string{"custom_*"},
			Custom:   []string{"special.file"},
		},
	}

	pm := NewPatternMatcher(config)

	// Test custom critical pattern
	shouldExclude, risk, _ := pm.ShouldExclude("/home/user/secret.custom")
	if !shouldExclude || risk != RiskLevelCritical {
		t.Errorf("Custom critical pattern not working: shouldExclude=%v, risk=%v", shouldExclude, risk)
	}

	// Test custom high pattern
	shouldExclude, risk, _ = pm.ShouldExclude("/home/user/custom_file")
	if !shouldExclude || risk != RiskLevelHigh {
		t.Errorf("Custom high pattern not working: shouldExclude=%v, risk=%v", shouldExclude, risk)
	}

	// Test custom pattern (should be treated as high risk)
	shouldExclude, risk, _ = pm.ShouldExclude("/home/user/special.file")
	if !shouldExclude || risk != RiskLevelHigh {
		t.Errorf("Custom pattern not working: shouldExclude=%v, risk=%v", shouldExclude, risk)
	}
}

func BenchmarkPatternMatcher_ShouldExclude(b *testing.B) {
	config := DefaultSecurityConfig()
	pm := NewPatternMatcher(config)
	testPath := "/home/user/.ssh/id_rsa"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.ShouldExclude(testPath)
	}
}
