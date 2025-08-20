package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_ScanFile(t *testing.T) {
	config := DefaultSecurityConfig()
	scanner := NewScanner(config)

	tests := []struct {
		name             string
		filePath         string
		content          string
		expectedPassed   bool
		expectedRisk     RiskLevel
		expectedFindings int
	}{
		{
			name:             "safe file",
			filePath:         "safe.txt",
			content:          "This is a safe configuration file with no secrets.",
			expectedPassed:   true,
			expectedRisk:     RiskLevelNone,
			expectedFindings: 0,
		},
		{
			name:             "file with API key",
			filePath:         "config.yml",
			content:          `api_key: "sk-1234567890abcdefghijklmnopqrstuvwxyz"`,
			expectedPassed:   false,
			expectedRisk:     RiskLevelHigh,
			expectedFindings: 1,
		},
		{
			name:             "env file (excluded by pattern)",
			filePath:         ".env",
			content:          `API_KEY=secret123`,
			expectedPassed:   false,
			expectedRisk:     RiskLevelCritical,
			expectedFindings: 1,
		},
		{
			name:             "private key file",
			filePath:         "private.key",
			content:          "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----",
			expectedPassed:   false,
			expectedRisk:     RiskLevelCritical,
			expectedFindings: 1,
		},
		{
			name:     "multiple secrets",
			filePath: "secrets.conf",
			content: `database_url = "postgres://user:password@localhost/db"
api_key = "sk-abcdefghijklmnopqrstuvwxyz123456"
jwt_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123"`,
			expectedPassed:   false,
			expectedRisk:     RiskLevelCritical, // JWT and API keys are CRITICAL risk
			expectedFindings: 4,                 // Enhanced detection finds more patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanner.ScanFile(tt.filePath, []byte(tt.content))
			if err != nil {
				t.Fatalf("ScanFile() error = %v", err)
			}

			if result.Passed != tt.expectedPassed {
				t.Errorf("ScanFile() passed = %v, want %v", result.Passed, tt.expectedPassed)
			}

			if result.Risk != tt.expectedRisk {
				t.Errorf("ScanFile() risk = %v, want %v", result.Risk, tt.expectedRisk)
			}

			if len(result.Findings) != tt.expectedFindings {
				t.Errorf("ScanFile() findings = %d, want %d", len(result.Findings), tt.expectedFindings)
			}
		})
	}
}

func TestScanner_ScanDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"safe.txt":    "This is a safe file.",
		".env":        "API_KEY=secret123\nDATABASE_URL=postgres://user:pass@localhost/db",
		"config.yml":  `database:\n  host: localhost\n  user: admin\n  password: secret_password_123`,
		".gitconfig":  `[user]\n  name = Test User\n  email = test@example.com`,
		"README.md":   "# This is a readme file",
		".ssh/id_rsa": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tempDir, relPath)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	config := DefaultSecurityConfig()
	scanner := NewScanner(config)

	tests := []struct {
		name        string
		ignores     []string
		minFindings int
		maxRisk     RiskLevel
	}{
		{
			name:        "scan all files",
			ignores:     []string{},
			minFindings: 3, // .env, config.yml, .ssh/id_rsa should be flagged
			maxRisk:     RiskLevelCritical,
		},
		{
			name:        "ignore .env files",
			ignores:     []string{".env"},
			minFindings: 2, // config.yml, .ssh/id_rsa should be flagged
			maxRisk:     RiskLevelCritical,
		},
		{
			name:        "ignore all risky files",
			ignores:     []string{".env", "*.yml", ".ssh/*"},
			minFindings: 0, // No findings expected since test@example.com is a placeholder
			maxRisk:     RiskLevelNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := scanner.ScanDirectory(tempDir, tt.ignores)
			if err != nil {
				t.Fatalf("ScanDirectory() error = %v", err)
			}

			if report.TotalFindings < tt.minFindings {
				t.Errorf("ScanDirectory() findings = %d, want at least %d",
					report.TotalFindings, tt.minFindings)
			}

			if report.HighestRisk > tt.maxRisk {
				t.Errorf("ScanDirectory() highest risk = %v, want max %v",
					report.HighestRisk, tt.maxRisk)
			}

			// Log details for debugging
			t.Logf("Scan results: %d files scanned, %d findings, highest risk: %s",
				report.ScannedFiles, report.TotalFindings, report.HighestRisk)

			for _, result := range report.Results {
				for _, finding := range result.Findings {
					t.Logf("  - %s: %s (%s, %.0f%%)",
						finding.Location.FilePath, finding.Type, finding.RiskLevel, finding.Confidence*100)
				}
			}
		})
	}
}

func TestScanner_SetSensitivity(t *testing.T) {
	config := DefaultSecurityConfig()
	scanner := NewScanner(config)

	// Test with a borderline secret that might be detected differently at different sensitivity levels
	content := `# Borderline secret that might not be caught at low sensitivity
semi_secret_key = "abcd1234efgh5678ijkl9012mnop3456"`

	// Test low sensitivity
	scanner.SetSensitivity(SensitivityLow)
	result, err := scanner.ScanFile("test.conf", []byte(content))
	if err != nil {
		t.Fatalf("ScanFile() error = %v", err)
	}
	lowSensitivityFindings := len(result.Findings)

	// Test high sensitivity
	scanner.SetSensitivity(SensitivityHigh)
	result, err = scanner.ScanFile("test.conf", []byte(content))
	if err != nil {
		t.Fatalf("ScanFile() error = %v", err)
	}
	highSensitivityFindings := len(result.Findings)

	// High sensitivity should find at least as many findings as low sensitivity
	if highSensitivityFindings < lowSensitivityFindings {
		t.Errorf("High sensitivity found fewer findings (%d) than low sensitivity (%d)",
			highSensitivityFindings, lowSensitivityFindings)
	}

	t.Logf("Low sensitivity: %d findings, High sensitivity: %d findings",
		lowSensitivityFindings, highSensitivityFindings)
}

func TestScanner_DisabledSecurity(t *testing.T) {
	config := DefaultSecurityConfig()
	config.Enabled = false
	scanner := NewScanner(config)

	// Even with secrets, disabled security should return empty results
	content := `api_key = "sk-1234567890abcdefghijklmnopqrstuvwxyz"
password = "super_secret_password"`

	result, err := scanner.ScanFile("config.yml", []byte(content))
	if err != nil {
		t.Fatalf("ScanFile() error = %v", err)
	}

	if len(result.Findings) != 0 {
		t.Errorf("Disabled security should find 0 secrets, found %d", len(result.Findings))
	}

	if result.Risk != RiskLevelNone {
		t.Errorf("Disabled security should have no risk, got %v", result.Risk)
	}

	if !result.Passed {
		t.Errorf("Disabled security should always pass")
	}
}

func TestScanReport_Summary(t *testing.T) {
	report := &ScanReport{
		ScannedFiles:  10,
		SkippedFiles:  2,
		TotalFindings: 5,
		HighestRisk:   RiskLevelHigh,
	}

	summary := report.Summary()
	expected := "Scanned 10 files (skipped 2): 5 findings, highest risk: HIGH"

	if summary != expected {
		t.Errorf("Summary() = %q, want %q", summary, expected)
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "text content",
			content:  []byte("This is plain text content"),
			expected: false,
		},
		{
			name:     "binary content with null bytes",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: true,
		},
		{
			name:     "mixed content with null byte",
			content:  []byte("text\x00binary"),
			expected: true,
		},
		{
			name:     "UTF-8 content",
			content:  []byte("Hello 世界"),
			expected: false,
		},
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinary(tt.content)
			if result != tt.expected {
				t.Errorf("isBinary() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskLevelNone, "NONE"},
		{RiskLevelLow, "LOW"},
		{RiskLevelMedium, "MEDIUM"},
		{RiskLevelHigh, "HIGH"},
		{RiskLevelCritical, "CRITICAL"},
		{RiskLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("RiskLevel.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFinding_Redact(t *testing.T) {
	tests := []struct {
		name     string
		finding  Finding
		expected string
	}{
		{
			name: "short value",
			finding: Finding{
				RawValue: "short",
			},
			expected: "********",
		},
		{
			name: "medium value",
			finding: Finding{
				RawValue: "sk-abcdefghijklmnop",
			},
			expected: "sk-a****mnop",
		},
		{
			name: "long value",
			finding: Finding{
				RawValue: "very_long_secret_key_that_should_be_redacted_properly",
			},
			expected: "very_l****operly",
		},
		{
			name: "empty value",
			finding: Finding{
				RawValue: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.finding.Redact()
			if result != tt.expected {
				t.Errorf("Finding.Redact() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func BenchmarkScanner_ScanFile(b *testing.B) {
	config := DefaultSecurityConfig()
	scanner := NewScanner(config)

	content := `# Sample configuration file
api_key = "sk-1234567890abcdefghijklmnopqrstuvwxyz"
database_url = "postgres://user:password@localhost:5432/db"
debug = true
log_level = "info"
jwt_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123"
normal_setting = "just a regular value"
another_api_key = "ak-0987654321zyxwvutsrqponmlkjihgfedcba"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanFile("test.conf", []byte(content))
	}
}

func BenchmarkScanner_ScanDirectory(b *testing.B) {
	// Create a temporary directory with test files
	tempDir := b.TempDir()

	files := map[string]string{
		"config1.yml": `api_key: "sk-1234567890abcdefghijklmnopqrstuvwxyz"`,
		"config2.yml": `database_url: "postgres://user:pass@localhost/db"`,
		"safe.txt":    "This is a safe file with no secrets.",
		"README.md":   "# Documentation file",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tempDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	config := DefaultSecurityConfig()
	scanner := NewScanner(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanDirectory(tempDir, []string{})
	}
}
