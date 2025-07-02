package security

import (
	"fmt"
	"os"
	"path/filepath"
)

type RiskLevel int

const (
	RiskLevelNone RiskLevel = iota
	RiskLevelLow
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLevelNone:
		return "NONE"
	case RiskLevelLow:
		return "LOW"
	case RiskLevelMedium:
		return "MEDIUM"
	case RiskLevelHigh:
		return "HIGH"
	case RiskLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

type SecretType string

const (
	SecretTypeAPIKey        SecretType = "api_key"
	SecretTypeAnthropicKey  SecretType = "anthropic_key"
	SecretTypeGenericAPIKey SecretType = "generic_api_key"
	SecretTypePassword      SecretType = "password"
	SecretTypePrivateKey    SecretType = "private_key"
	SecretTypeToken         SecretType = "token"
	SecretTypeAWSKey        SecretType = "aws_key"
	SecretTypeGitHubToken   SecretType = "github_token"
	SecretTypeJWT           SecretType = "jwt"
	SecretTypeDatabaseURL   SecretType = "database_url"
	SecretTypeGeneric       SecretType = "generic_secret"
	SecretTypePII           SecretType = "pii"
	SecretTypeCreditCard    SecretType = "credit_card"
	SecretTypeSSN           SecretType = "ssn"
	SecretTypeEmail         SecretType = "email"
	SecretTypeIPAddress     SecretType = "ip_address"
)

type Location struct {
	FilePath   string
	LineNumber int
	Column     int
	LineText   string
}

type Finding struct {
	Type       SecretType
	Value      string // Redacted version
	RawValue   string // Original value (kept for internal use)
	Location   Location
	Confidence float64  // 0.0 to 1.0
	Context    string   // Surrounding text
	Reasons    []string // Why it was flagged
	RiskLevel  RiskLevel
}

func (f *Finding) Redact() string {
	if len(f.RawValue) == 0 {
		return ""
	}

	// Show first few and last few characters
	if len(f.RawValue) <= 8 {
		return "********"
	}

	visible := 4
	if len(f.RawValue) > 20 {
		visible = 6
	}

	return f.RawValue[:visible] + "****" + f.RawValue[len(f.RawValue)-visible:]
}

type ScanResult struct {
	FilePath string
	Findings []Finding
	Risk     RiskLevel
	Passed   bool
	Error    error
}

type ScanReport struct {
	Results       []ScanResult
	TotalFiles    int
	ScannedFiles  int
	SkippedFiles  int
	TotalFindings int
	HighestRisk   RiskLevel
}

type Scanner struct {
	config          *SecurityConfig
	patternMatcher  *PatternMatcher
	contentDetector *Detector
}

func NewScanner(config *SecurityConfig) *Scanner {
	return &Scanner{
		config:          config,
		patternMatcher:  NewPatternMatcher(config),
		contentDetector: NewDetector(config),
	}
}

func (s *Scanner) ScanFile(path string, content []byte) (*ScanResult, error) {
	result := &ScanResult{
		FilePath: path,
		Findings: []Finding{},
		Risk:     RiskLevelNone,
		Passed:   true,
	}

	// Return early if security scanning is disabled
	if !s.config.Enabled {
		return result, nil
	}

	// First check if file should be excluded by pattern
	if s.config.ExcludePatterns {
		shouldExclude, risk, reason := s.patternMatcher.ShouldExclude(path)
		if shouldExclude && risk >= RiskLevelHigh {
			result.Risk = risk
			result.Passed = false
			result.Findings = append(result.Findings, Finding{
				Type:       SecretTypeGeneric,
				Location:   Location{FilePath: path},
				Confidence: 1.0,
				Reasons:    []string{reason},
				RiskLevel:  risk,
			})
			return result, nil
		}
	}

	// Then scan content if enabled
	if s.config.ScanContent && len(content) > 0 {
		// Skip binary files unless configured to scan them
		if !s.config.ContentScan.ScanBinaryFiles && isBinary(content) {
			return result, nil
		}

		// Skip files that are too large
		if s.config.ContentScan.MaxFileSize > 0 && len(content) > s.config.ContentScan.MaxFileSize {
			return result, nil
		}

		findings := s.contentDetector.DetectSecrets(content, path)
		result.Findings = append(result.Findings, findings...)

		// Update risk level based on findings
		for _, finding := range findings {
			if finding.RiskLevel > result.Risk {
				result.Risk = finding.RiskLevel
			}
		}

		result.Passed = result.Risk < RiskLevelHigh || (result.Risk == RiskLevelHigh && !s.config.FailOnSecrets)
	}

	return result, nil
}

func (s *Scanner) ScanDirectory(path string, ignores []string) (*ScanReport, error) {
	report := &ScanReport{
		Results:       []ScanResult{},
		TotalFiles:    0,
		ScannedFiles:  0,
		SkippedFiles:  0,
		TotalFindings: 0,
		HighestRisk:   RiskLevelNone,
	}

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be ignored
			for _, ignore := range ignores {
				if matched, _ := filepath.Match(ignore, info.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		report.TotalFiles++

		// Check if file should be ignored
		for _, ignore := range ignores {
			// Check against basename
			if matched, _ := filepath.Match(ignore, filepath.Base(filePath)); matched {
				report.SkippedFiles++
				return nil
			}
			// Check against relative path from scan root
			relPath, _ := filepath.Rel(path, filePath)
			if matched, _ := filepath.Match(ignore, relPath); matched {
				report.SkippedFiles++
				return nil
			}
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			report.SkippedFiles++
			return nil
		}

		// Scan the file
		result, err := s.ScanFile(filePath, content)
		if err != nil {
			result.Error = err
		}

		report.ScannedFiles++
		report.Results = append(report.Results, *result)
		report.TotalFindings += len(result.Findings)

		if result.Risk > report.HighestRisk {
			report.HighestRisk = result.Risk
		}

		return nil
	})

	return report, err
}

func (s *Scanner) SetSensitivity(level SensitivityLevel) {
	s.config.Sensitivity = level
	// Update detector sensitivity
	s.contentDetector.SetSensitivity(level)
}

func isBinary(content []byte) bool {
	// Simple heuristic: check for null bytes in first 8192 bytes
	checkLen := len(content)
	if checkLen > 8192 {
		checkLen = 8192
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

func (r *ScanReport) Summary() string {
	return fmt.Sprintf(
		"Scanned %d files (skipped %d): %d findings, highest risk: %s",
		r.ScannedFiles,
		r.SkippedFiles,
		r.TotalFindings,
		r.HighestRisk,
	)
}
