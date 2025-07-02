package security

import "strings"

type SecurityConfig struct {
	// Global security settings
	Enabled         bool             `toml:"enabled"`
	ScanContent     bool             `toml:"scan_content"`
	ExcludePatterns bool             `toml:"exclude_patterns"`
	Sensitivity     SensitivityLevel `toml:"sensitivity"`
	FailOnSecrets   bool             `toml:"fail_on_secrets"`
	Interactive     bool             `toml:"interactive"`

	// Pattern exclusion configuration
	PatternConfig PatternConfig `toml:"exclude_patterns"`

	// Content scanning configuration
	ContentScan ContentScanConfig `toml:"content_scan"`

	// Allowlist configuration
	Allowlist AllowlistConfig `toml:"allowlist"`
}

type PatternConfig struct {
	Critical []string `toml:"critical"`
	High     []string `toml:"high"`
	Medium   []string `toml:"medium"`
	Low      []string `toml:"low"`
	Custom   []string `toml:"custom"`
}

type ContentScanConfig struct {
	EntropyThreshold float64 `toml:"entropy_threshold"`
	MinSecretLength  int     `toml:"min_secret_length"`
	MaxFileSize      int     `toml:"max_file_size"`
	ScanBinaryFiles  bool    `toml:"scan_binary_files"`
	ContextWindow    int     `toml:"context_window"`
}

type AllowlistConfig struct {
	Patterns []string `toml:"patterns"`
	Files    []string `toml:"files"`
}

// DefaultSecurityConfig returns a security configuration with sensible defaults
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Enabled:         true,
		ScanContent:     true,
		ExcludePatterns: true,
		Sensitivity:     SensitivityMedium,
		FailOnSecrets:   true,
		Interactive:     true,

		PatternConfig: PatternConfig{
			Critical: []string{},
			High:     []string{},
			Medium:   []string{},
			Low:      []string{},
			Custom:   []string{},
		},

		ContentScan: ContentScanConfig{
			EntropyThreshold: MediumEntropyThreshold,
			MinSecretLength:  20,
			MaxFileSize:      10 * 1024 * 1024, // 10MB
			ScanBinaryFiles:  false,
			ContextWindow:    50,
		},

		Allowlist: AllowlistConfig{
			Patterns: []string{"EXAMPLE_*", "DEMO_*", "TEST_*"},
			Files:    []string{},
		},
	}
}

// Merge combines two security configurations, with the second taking precedence
func (sc *SecurityConfig) Merge(other *SecurityConfig) {
	if other == nil {
		return
	}

	// Merge basic settings
	if other.Enabled != sc.Enabled {
		sc.Enabled = other.Enabled
	}
	if other.ScanContent != sc.ScanContent {
		sc.ScanContent = other.ScanContent
	}
	if other.ExcludePatterns != sc.ExcludePatterns {
		sc.ExcludePatterns = other.ExcludePatterns
	}
	if other.Sensitivity != "" {
		sc.Sensitivity = other.Sensitivity
	}
	if other.FailOnSecrets != sc.FailOnSecrets {
		sc.FailOnSecrets = other.FailOnSecrets
	}
	if other.Interactive != sc.Interactive {
		sc.Interactive = other.Interactive
	}

	// Merge pattern configs
	if len(other.PatternConfig.Critical) > 0 {
		sc.PatternConfig.Critical = append(sc.PatternConfig.Critical, other.PatternConfig.Critical...)
	}
	if len(other.PatternConfig.High) > 0 {
		sc.PatternConfig.High = append(sc.PatternConfig.High, other.PatternConfig.High...)
	}
	if len(other.PatternConfig.Medium) > 0 {
		sc.PatternConfig.Medium = append(sc.PatternConfig.Medium, other.PatternConfig.Medium...)
	}
	if len(other.PatternConfig.Low) > 0 {
		sc.PatternConfig.Low = append(sc.PatternConfig.Low, other.PatternConfig.Low...)
	}
	if len(other.PatternConfig.Custom) > 0 {
		sc.PatternConfig.Custom = append(sc.PatternConfig.Custom, other.PatternConfig.Custom...)
	}

	// Merge content scan config
	if other.ContentScan.EntropyThreshold != 0 {
		sc.ContentScan.EntropyThreshold = other.ContentScan.EntropyThreshold
	}
	if other.ContentScan.MinSecretLength != 0 {
		sc.ContentScan.MinSecretLength = other.ContentScan.MinSecretLength
	}
	if other.ContentScan.MaxFileSize != 0 {
		sc.ContentScan.MaxFileSize = other.ContentScan.MaxFileSize
	}
	if other.ContentScan.ScanBinaryFiles != sc.ContentScan.ScanBinaryFiles {
		sc.ContentScan.ScanBinaryFiles = other.ContentScan.ScanBinaryFiles
	}
	if other.ContentScan.ContextWindow != 0 {
		sc.ContentScan.ContextWindow = other.ContentScan.ContextWindow
	}

	// Merge allowlist config
	if len(other.Allowlist.Patterns) > 0 {
		sc.Allowlist.Patterns = append(sc.Allowlist.Patterns, other.Allowlist.Patterns...)
	}
	if len(other.Allowlist.Files) > 0 {
		sc.Allowlist.Files = append(sc.Allowlist.Files, other.Allowlist.Files...)
	}
}

// Validate checks if the security configuration is valid
func (sc *SecurityConfig) Validate() error {
	if sc.ContentScan.EntropyThreshold < 0 || sc.ContentScan.EntropyThreshold > 8 {
		return &ValidationError{Field: "entropy_threshold", Message: "must be between 0 and 8"}
	}

	if sc.ContentScan.MaxFileSize < 0 {
		return &ValidationError{Field: "max_file_size", Message: "must be non-negative"}
	}

	if sc.ContentScan.ContextWindow < 0 {
		return &ValidationError{Field: "context_window", Message: "must be non-negative"}
	}

	validSensitivities := []SensitivityLevel{
		SensitivityLow, SensitivityMedium, SensitivityHigh, SensitivityParanoid,
	}

	validSensitivity := false
	for _, valid := range validSensitivities {
		if sc.Sensitivity == valid {
			validSensitivity = true
			break
		}
	}

	if !validSensitivity {
		return &ValidationError{
			Field:   "sensitivity",
			Message: "must be one of: low, medium, high, paranoid",
		}
	}

	if sc.ContentScan.MinSecretLength < 1 {
		return &ValidationError{Field: "min_secret_length", Message: "must be positive"}
	}

	return nil
}

// IsFileAllowed checks if a file is in the allowlist
func (sc *SecurityConfig) IsFileAllowed(filePath string) bool {
	for _, allowedFile := range sc.Allowlist.Files {
		if filePath == allowedFile {
			return true
		}
	}
	return false
}

// IsPatternAllowed checks if a value matches any allowlist pattern
func (sc *SecurityConfig) IsPatternAllowed(value string) bool {
	for _, pattern := range sc.Allowlist.Patterns {
		// Simple glob-like matching
		if matchesPattern(value, pattern) {
			return true
		}
	}
	return false
}

func matchesPattern(value, pattern string) bool {
	// Simple implementation - can be enhanced with full glob support
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	}

	return value == pattern
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error in field '" + e.Field + "': " + e.Message
}
