package security

import (
	"math"
	"testing"
)

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		delta    float64
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "single character",
			input:    "a",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "repeated character",
			input:    "aaaa",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "balanced binary",
			input:    "0101",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "random-like string",
			input:    "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: 5.0,
			delta:    1.0,
		},
		{
			name:     "high entropy API key",
			input:    "AKIA1234567890ABCDEF",
			expected: 3.5,
			delta:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateEntropy(tt.input)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("CalculateEntropy(%q) = %f, want %f ± %f", tt.input, result, tt.expected, tt.delta)
			}
		})
	}
}

func TestEntropyAnalyzer_IsHighEntropy(t *testing.T) {
	analyzer := NewEntropyAnalyzer(MediumEntropyThreshold)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "short string",
			input:    "short",
			expected: false,
		},
		{
			name:     "very long string",
			input:    string(make([]byte, MaxEntropyLength+1)),
			expected: false,
		},
		{
			name:     "low entropy long string",
			input:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: false,
		},
		{
			name:     "high entropy API key",
			input:    "sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEF",
			expected: true,
		},
		{
			name:     "AWS access key",
			input:    "AKIA1234567890ABCDEF",
			expected: true, // AWS keys with AKIA prefix should be detected
		},
		{
			name:     "random-like token",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: true, // JWT tokens should be detected with prefix boost
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.IsHighEntropy(tt.input)
			if result != tt.expected {
				t.Errorf("IsHighEntropy(%q) = %v, want %v (entropy: %.2f)",
					tt.input, result, tt.expected, CalculateEntropy(tt.input))
			}
		})
	}
}

func TestEntropyAnalyzer_AnalyzeString(t *testing.T) {
	analyzer := NewEntropyAnalyzer(MediumEntropyThreshold)

	tests := []struct {
		name          string
		input         string
		expectSecret  bool
		minConfidence float64
	}{
		{
			name:          "short string",
			input:         "short",
			expectSecret:  false,
			minConfidence: 0.0,
		},
		{
			name:          "high entropy secret",
			input:         "sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEF",
			expectSecret:  true, // Should be detected with fixed logic
			minConfidence: 0.7,
		},
		{
			name:          "placeholder string",
			input:         "example_api_key_1234567890abcdef",
			expectSecret:  false,
			minConfidence: 0.0,
		},
		{
			name:          "repeated pattern",
			input:         "abcabcabcabcabcabcabcabcabc",
			expectSecret:  false,
			minConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzer.AnalyzeString(tt.input)

			if analysis.IsSecret != tt.expectSecret {
				t.Errorf("AnalyzeString(%q).IsSecret = %v, want %v", tt.input, analysis.IsSecret, tt.expectSecret)
			}

			if analysis.Confidence < tt.minConfidence {
				t.Errorf("AnalyzeString(%q).Confidence = %f, want >= %f", tt.input, analysis.Confidence, tt.minConfidence)
			}
		})
	}
}

func TestEntropyAnalyzer_ExtractHighEntropyStrings(t *testing.T) {
	analyzer := NewEntropyAnalyzer(MediumEntropyThreshold)

	tests := []struct {
		name     string
		input    string
		expected int // minimum expected high-entropy strings
	}{
		{
			name:     "no secrets",
			input:    "This is a normal text file with no secrets.",
			expected: 0,
		},
		{
			name:     "API key in config",
			input:    `api_key="sk-1234567890abcdefghijklmnopqrstuvwxyz"`,
			expected: 1,
		},
		{
			name: "multiple secrets",
			input: `API_KEY=sk-abc123def456ghi789jkl
			           TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
			           PASSWORD=very_secure_password_123`,
			expected: 2, // Both API key and JWT token should be detected
		},
		{
			name: "mixed content",
			input: `# Configuration file
			           database_host=localhost
			           api_key=sk-1234567890abcdefghijklmnopqr
			           debug=true`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.ExtractHighEntropyStrings(tt.input)
			if len(result) < tt.expected {
				t.Errorf("ExtractHighEntropyStrings(%q) found %d strings, want at least %d. Found: %v",
					tt.input, len(result), tt.expected, result)
			}
		})
	}
}

func BenchmarkCalculateEntropy(b *testing.B) {
	testString := "sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEF1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateEntropy(testString)
	}
}

func BenchmarkEntropyAnalyzer_IsHighEntropy(b *testing.B) {
	analyzer := NewEntropyAnalyzer(MediumEntropyThreshold)
	testString := "sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEF1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.IsHighEntropy(testString)
	}
}
