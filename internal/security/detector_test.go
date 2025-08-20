package security

import (
	"testing"
)

func TestDetector_DetectSecrets(t *testing.T) {
	config := DefaultSecurityConfig()
	detector := NewDetector(config)

	tests := []struct {
		name                string
		content             string
		expectedFindings    int
		expectedSecretTypes []SecretType
		minConfidence       float64
	}{
		{
			name:             "no secrets",
			content:          "This is a normal configuration file with no secrets.",
			expectedFindings: 0,
		},
		{
			name:                "API key pattern",
			content:             `api_key = "sk-1234567890abcdefghijklmnopqrstuvwxyz"`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypeAPIKey},
			minConfidence:       0.7,
		},
		{
			name:                "AWS access key",
			content:             `AWS_ACCESS_KEY_ID=AKIA1234567890ABCDEF`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypeAWSKey},
			minConfidence:       0.8,
		},
		{
			name:                "JWT token",
			content:             `token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`,
			expectedFindings:    2, // Pattern match + context match (after security enhancements)
			expectedSecretTypes: []SecretType{SecretTypeJWT},
			minConfidence:       0.8,
		},
		{
			name: "Private key",
			content: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdefghijklmnopqrstuvwxyz
-----END RSA PRIVATE KEY-----`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypePrivateKey},
			minConfidence:       0.9,
		},
		{
			name:                "Password",
			content:             `password = "my_super_secret_password_123"`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypePassword},
			minConfidence:       0.6,
		},
		{
			name:                "Database URL",
			content:             `DATABASE_URL=postgres://user:password@localhost:5432/db`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypeDatabaseURL},
			minConfidence:       0.8,
		},
		{
			name:                "Email address",
			content:             `admin_email = "admin@company.com"`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypeEmail},
			minConfidence:       0.5,
		},
		{
			name:                "Credit card number",
			content:             `credit_card = "4111111111111111"`,
			expectedFindings:    1,
			expectedSecretTypes: []SecretType{SecretTypeCreditCard},
			minConfidence:       0.8,
		},
		{
			name: "Multiple secrets",
			content: `# Configuration
api_key = "sk-abcdefghijklmnopqrstuvwxyz123456"
password = "super_secret_password"
DATABASE_URL = "postgres://user:pass@localhost/db"`,
			expectedFindings: 3,
			expectedSecretTypes: []SecretType{
				SecretTypeAPIKey,
				SecretTypePassword,
				SecretTypeDatabaseURL,
			},
			minConfidence: 0.6,
		},
		{
			name: "High entropy string",
			content: `# This looks like a secret
random_token = "aB3dE6fG9hI2jK5lM8nO1pQ4rS7tU0vW"`,
			expectedFindings: 1, // Should be detected by entropy analysis
			minConfidence:    0.5,
		},
		{
			name:             "Placeholder values",
			content:          `api_key = "your_api_key_here"\npassword = "example_password"`,
			expectedFindings: 0, // Should be filtered out as placeholders
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := detector.DetectSecrets([]byte(tt.content), "test.conf")

			if len(findings) != tt.expectedFindings {
				t.Errorf("DetectSecrets() found %d secrets, want %d. Findings: %v",
					len(findings), tt.expectedFindings, findings)
			}

			if tt.expectedFindings > 0 {
				// Check that we found the expected secret types
				foundTypes := make(map[SecretType]bool)
				for _, finding := range findings {
					foundTypes[finding.Type] = true

					// Check minimum confidence
					if finding.Confidence < tt.minConfidence {
						t.Errorf("Finding confidence %.2f below minimum %.2f for type %s",
							finding.Confidence, tt.minConfidence, finding.Type)
					}
				}

				for _, expectedType := range tt.expectedSecretTypes {
					if !foundTypes[expectedType] {
						t.Errorf("Expected secret type %s not found", expectedType)
					}
				}
			}
		})
	}
}

func TestDetector_isPlaceholder(t *testing.T) {
	detector := NewDetector(nil)

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "example placeholder",
			value:    "example_api_key",
			expected: true,
		},
		{
			name:     "test placeholder",
			value:    "test_password_123",
			expected: true,
		},
		{
			name:     "your placeholder",
			value:    "your_secret_here",
			expected: true,
		},
		{
			name:     "repeated characters",
			value:    "aaaaaaaaaaaaaaaa",
			expected: true,
		},
		{
			name:     "asterisks",
			value:    "****************",
			expected: true,
		},
		{
			name:     "real secret",
			value:    "sk-1234567890abcdefghijklmnop",
			expected: false,
		},
		{
			name:     "random string",
			value:    "aB3dE6fG9hI2jK5l",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isPlaceholder(tt.value)
			if result != tt.expected {
				t.Errorf("isPlaceholder(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestDetector_extractSecretCandidates(t *testing.T) {
	detector := NewDetector(nil)

	tests := []struct {
		name             string
		line             string
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "simple key-value",
			line:          `api_key = "sk-1234567890abcdef"`,
			expectedCount: 1,
			shouldContain: []string{"sk-1234567890abcdef"},
		},
		{
			name:          "JSON format",
			line:          `{"api_key": "secret123", "token": "abc456"}`,
			expectedCount: 2,
			shouldContain: []string{"secret123", "abc456"},
		},
		{
			name:          "environment variable",
			line:          `export API_KEY="very_secret_key_here"`,
			expectedCount: 1,
			shouldContain: []string{"very_secret_key_here"},
		},
		{
			name:             "no secrets",
			line:             "This is just a comment line",
			expectedCount:    0,
			shouldNotContain: []string{"comment", "line"},
		},
		{
			name:          "multiple formats",
			line:          `password="secret1" token='secret2' key=secret3`,
			expectedCount: 3,
			shouldContain: []string{"secret1", "secret2", "secret3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := detector.extractSecretCandidates(tt.line)

			if len(candidates) < tt.expectedCount {
				t.Errorf("extractSecretCandidates(%q) found %d candidates, want at least %d",
					tt.line, len(candidates), tt.expectedCount)
			}

			for _, should := range tt.shouldContain {
				found := false
				for _, candidate := range candidates {
					if candidate == should {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected candidate %q not found in %v", should, candidates)
				}
			}

			for _, shouldNot := range tt.shouldNotContain {
				for _, candidate := range candidates {
					if candidate == shouldNot {
						t.Errorf("Unexpected candidate %q found in %v", shouldNot, candidates)
					}
				}
			}
		})
	}
}

func TestDetector_SetSensitivity(t *testing.T) {
	detector := NewDetector(nil)

	tests := []struct {
		name              string
		sensitivity       SensitivityLevel
		expectedThreshold float64
	}{
		{
			name:              "low sensitivity",
			sensitivity:       SensitivityLow,
			expectedThreshold: LowEntropyThreshold,
		},
		{
			name:              "medium sensitivity",
			sensitivity:       SensitivityMedium,
			expectedThreshold: MediumEntropyThreshold,
		},
		{
			name:              "high sensitivity",
			sensitivity:       SensitivityHigh,
			expectedThreshold: HighEntropyThreshold,
		},
		{
			name:              "paranoid sensitivity",
			sensitivity:       SensitivityParanoid,
			expectedThreshold: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector.SetSensitivity(tt.sensitivity)

			if detector.entropyAnalyzer.threshold != tt.expectedThreshold {
				t.Errorf("SetSensitivity(%s) threshold = %f, want %f",
					tt.sensitivity, detector.entropyAnalyzer.threshold, tt.expectedThreshold)
			}
		})
	}
}

func TestDetector_calculatePatternConfidence(t *testing.T) {
	detector := NewDetector(nil)

	tests := []struct {
		name          string
		secretType    SecretType
		value         string
		context       string
		minConfidence float64
		maxConfidence float64
	}{
		{
			name:          "private key",
			secretType:    SecretTypePrivateKey,
			value:         "-----BEGIN RSA PRIVATE KEY-----",
			context:       "private key file",
			minConfidence: 0.9,
			maxConfidence: 1.0,
		},
		{
			name:          "AWS key",
			secretType:    SecretTypeAWSKey,
			value:         "AKIA1234567890ABCDEF",
			context:       "AWS configuration",
			minConfidence: 0.8,
			maxConfidence: 1.0,
		},
		{
			name:          "placeholder password",
			secretType:    SecretTypePassword,
			value:         "example_password",
			context:       "config template",
			minConfidence: 0.0,
			maxConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := detector.calculatePatternConfidence(tt.secretType, tt.value, tt.context)

			if confidence < tt.minConfidence || confidence > tt.maxConfidence {
				t.Errorf("calculatePatternConfidence(%s, %q, %q) = %f, want between %f and %f",
					tt.secretType, tt.value, tt.context, confidence, tt.minConfidence, tt.maxConfidence)
			}
		})
	}
}

func TestExtractQuotedStrings(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "double quotes",
			line:     `api_key = "secret123"`,
			expected: []string{"secret123"},
		},
		{
			name:     "single quotes",
			line:     `password = 'secret456'`,
			expected: []string{"secret456"},
		},
		{
			name:     "mixed quotes",
			line:     `key1="value1" key2='value2'`,
			expected: []string{"value1", "value2"},
		},
		{
			name:     "no quotes",
			line:     "no quoted strings here",
			expected: []string{},
		},
		{
			name:     "empty quotes",
			line:     `empty="" also=''`,
			expected: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractQuotedStrings(tt.line)

			if len(result) != len(tt.expected) {
				t.Errorf("extractQuotedStrings(%q) = %v, want %v", tt.line, result, tt.expected)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("extractQuotedStrings(%q)[%d] = %q, want %q", tt.line, i, result[i], expected)
				}
			}
		})
	}
}

func TestRedactValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "short value",
			value:    "short",
			expected: "********",
		},
		{
			name:     "medium value",
			value:    "sk-abcdefghijklmnop",
			expected: "sk-a****mnop",
		},
		{
			name:     "long value",
			value:    "very_long_secret_key_that_should_be_redacted_properly",
			expected: "very_l****operly",
		},
		{
			name:     "empty value",
			value:    "",
			expected: "********",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactValue(tt.value)
			if result != tt.expected {
				t.Errorf("redactValue(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func BenchmarkDetector_DetectSecrets(b *testing.B) {
	config := DefaultSecurityConfig()
	detector := NewDetector(config)

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
		detector.DetectSecrets([]byte(content), "test.conf")
	}
}

func TestDetector_RealWorldExamples(t *testing.T) {
	config := DefaultSecurityConfig()
	detector := NewDetector(config)

	realWorldExamples := []struct {
		name           string
		content        string
		filename       string
		expectFindings bool
	}{
		{
			name: "bashrc with history",
			content: `# .bashrc
export PATH=$PATH:/usr/local/bin
alias ll='ls -la'
# Don't put sensitive stuff here!`,
			filename:       ".bashrc",
			expectFindings: false,
		},
		{
			name: "env file with secrets",
			content: `DATABASE_URL=postgres://user:secretpass@localhost/mydb
API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz
DEBUG=true
LOG_LEVEL=info`,
			filename:       ".env",
			expectFindings: true,
		},
		{
			name: "ssh config",
			content: `Host myserver
    HostName example.com
    User myuser
    Port 22
    IdentityFile ~/.ssh/id_rsa`,
			filename:       ".ssh/config",
			expectFindings: false,
		},
		{
			name: "git config with email",
			content: `[user]
    name = John Doe
    email = john.doe@company.com
[core]
    editor = vim`,
			filename:       ".gitconfig",
			expectFindings: true, // Email should be detected
		},
	}

	for _, example := range realWorldExamples {
		t.Run(example.name, func(t *testing.T) {
			findings := detector.DetectSecrets([]byte(example.content), example.filename)

			hasFindings := len(findings) > 0
			if hasFindings != example.expectFindings {
				t.Errorf("Real world example %q: expected findings=%v, got findings=%v (count: %d)",
					example.name, example.expectFindings, hasFindings, len(findings))

				if len(findings) > 0 {
					t.Logf("Findings:")
					for _, finding := range findings {
						t.Logf("  - %s: %s (%.0f%%)", finding.Type, finding.Value, finding.Confidence*100)
					}
				}
			}
		})
	}
}
