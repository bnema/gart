package security

import (
	"path/filepath"
	"regexp"
	"strings"
)

type PatternMatcher struct {
	patterns map[RiskLevel][]string
	compiled map[string]*regexp.Regexp
}

var defaultPatterns = map[RiskLevel][]string{
	RiskLevelCritical: {
		// Environment files
		".env",
		".env.*",
		"*.env",

		// Private keys
		"*.key",
		"*.pem",
		"*.p12",
		"*.pfx",
		"*.jks",
		"*.keystore",
		"*.truststore",

		// SSH keys
		".ssh/id_*",
		".ssh/*_rsa",
		".ssh/*_dsa",
		".ssh/*_ecdsa",
		".ssh/*_ed25519",

		// Cloud credentials
		".aws/credentials",
		".gcloud/credentials.db",
		".gcloud/legacy_credentials",
		".azure/credentials",

		// Password stores
		".password-store/*",
		".pass/*",
		"*.kdbx",
		"*.kdb",
		".1password/*",
		".bitwarden/*",
		".lastpass/*",

		// Terraform state
		"terraform.tfstate",
		"terraform.tfstate.backup",
		"*.tfvars",

		// Docker
		".docker/config.json",
		".dockercfg",

		// Kubernetes
		".kube/config",
		".kube/cache/*",
	},

	RiskLevelHigh: {
		// Shell history
		".*_history",
		".bash_history",
		".zsh_history",
		".fish_history",
		".python_history",
		".node_repl_history",
		".psql_history",
		".mysql_history",
		".lesshst",
		".viminfo",
		".wget-hsts",

		// Git credentials
		".git-credentials",
		".netrc",
		".authinfo",
		".authinfo.gpg",

		// Package managers
		".npmrc",
		".pypirc",
		".gem/credentials",
		".bundle/config",
		".cargo/credentials",
		".m2/settings.xml",

		// Database files
		"*.db",
		"*.sqlite",
		"*.sqlite3",
		"database.yml",
		"mongoid.yml",

		// SSH known hosts
		".ssh/known_hosts",
		".ssh/authorized_keys",

		// GPG
		".gnupg/secring.gpg",
		".gnupg/trustdb.gpg",
		".gnupg/random_seed",
	},

	RiskLevelMedium: {
		// Config files that might contain secrets
		".gitconfig",
		"config.yml",
		"config.yaml",
		"settings.yml",
		"settings.yaml",
		"secrets.yml",
		"secrets.yaml",
		"config/secrets.*",

		// Log files
		"*.log",
		"logs/*",

		// Cache directories
		".cache/*",
		"cache/*",
		".tmp/*",
		"tmp/*",
		"temp/*",

		// IDE files that might contain secrets
		".vscode/settings.json",
		".vscode/launch.json",
		".idea/workspace.xml",
		".idea/tasks.xml",

		// Backup files
		"*.bak",
		"*.backup",
		"*.old",
		"*.orig",
		"*~",
		".#*",

		// Test files that might have credentials
		".env.test",
		".env.development",
		"test.env",
		"dev.env",
	},

	RiskLevelLow: {
		// OS files
		".DS_Store",
		"Thumbs.db",
		"desktop.ini",
		".Spotlight-V100",
		".Trashes",
		".fseventsd",

		// General cache/temp
		"node_modules/*",
		".npm/*",
		".yarn/*",
		"vendor/*",
		"__pycache__/*",
		"*.pyc",

		// Build artifacts
		"dist/*",
		"build/*",
		"target/*",
		"*.o",
		"*.so",
		"*.dylib",
		"*.dll",
	},
}

func NewPatternMatcher(config *SecurityConfig) *PatternMatcher {
	pm := &PatternMatcher{
		patterns: make(map[RiskLevel][]string),
		compiled: make(map[string]*regexp.Regexp),
	}

	// If config is nil or ExcludePatterns is true, load default patterns
	if config == nil || config.ExcludePatterns {
		// Load default patterns
		for level, patterns := range defaultPatterns {
			pm.patterns[level] = append(pm.patterns[level], patterns...)
		}
	}

	// Add custom patterns from config
	if config != nil {
		if len(config.PatternConfig.Critical) > 0 {
			pm.patterns[RiskLevelCritical] = append(pm.patterns[RiskLevelCritical], config.PatternConfig.Critical...)
		}
		if len(config.PatternConfig.High) > 0 {
			pm.patterns[RiskLevelHigh] = append(pm.patterns[RiskLevelHigh], config.PatternConfig.High...)
		}
		if len(config.PatternConfig.Medium) > 0 {
			pm.patterns[RiskLevelMedium] = append(pm.patterns[RiskLevelMedium], config.PatternConfig.Medium...)
		}
		if len(config.PatternConfig.Low) > 0 {
			pm.patterns[RiskLevelLow] = append(pm.patterns[RiskLevelLow], config.PatternConfig.Low...)
		}
		if len(config.PatternConfig.Custom) > 0 {
			pm.patterns[RiskLevelHigh] = append(pm.patterns[RiskLevelHigh], config.PatternConfig.Custom...)
		}
	}

	return pm
}

func (pm *PatternMatcher) ShouldExclude(path string) (bool, RiskLevel, string) {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	baseName := filepath.Base(normalizedPath)

	// Check patterns in order of severity
	levels := []RiskLevel{RiskLevelCritical, RiskLevelHigh, RiskLevelMedium, RiskLevelLow}

	for _, level := range levels {
		patterns := pm.patterns[level]
		for _, pattern := range patterns {
			matched := pm.matchPattern(normalizedPath, baseName, pattern)
			if matched {
				return true, level, pattern
			}
		}
	}

	return false, RiskLevelNone, ""
}

func (pm *PatternMatcher) matchPattern(fullPath, baseName, pattern string) bool {
	// Handle different pattern types

	// Direct filename match
	if pattern == baseName {
		return true
	}

	// Glob pattern match for basename
	if matched, _ := filepath.Match(pattern, baseName); matched {
		return true
	}

	// Glob pattern match for full path
	if matched, _ := filepath.Match(pattern, fullPath); matched {
		return true
	}

	// Directory pattern (ends with /*)
	if strings.HasSuffix(pattern, "/*") {
		dirPattern := strings.TrimSuffix(pattern, "/*")
		if strings.Contains(fullPath, dirPattern+"/") {
			return true
		}
	}

	// Prefix pattern (starts with directory)
	if strings.Contains(pattern, "/") {
		// For patterns like .ssh/id_*, we need to check if the pattern matches the end of the path
		// Extract the directory and file pattern parts
		parts := strings.Split(pattern, "/")
		if len(parts) == 2 {
			dirPart := parts[0]
			filePart := parts[1]

			// Check if the path contains the directory part
			if strings.Contains(fullPath, "/"+dirPart+"/") || strings.Contains(fullPath, dirPart+"/") {
				// Extract the filename from the full path after the directory part
				idx := strings.LastIndex(fullPath, dirPart+"/")
				if idx != -1 {
					remainingPath := fullPath[idx+len(dirPart)+1:]
					// Check if the file pattern matches
					if matched, _ := filepath.Match(filePart, remainingPath); matched {
						return true
					}
					// Also check just the basename
					if matched, _ := filepath.Match(filePart, baseName); matched {
						return true
					}
				}
			}
		}

		// Try glob match on the full path first (for patterns like .ssh/id_*)
		if matched, _ := filepath.Match("*/"+pattern, fullPath); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, fullPath); matched {
			return true
		}

		// Exact prefix match
		if strings.HasPrefix(fullPath, pattern) {
			return true
		}
		if strings.Contains(fullPath, "/"+pattern) {
			return true
		}
	}

	// Regex pattern (if it starts with ^ or contains regex metacharacters)
	if pm.isRegexPattern(pattern) {
		if compiled, exists := pm.compiled[pattern]; exists {
			return compiled.MatchString(fullPath) || compiled.MatchString(baseName)
		} else {
			if compiled, err := regexp.Compile(pattern); err == nil {
				pm.compiled[pattern] = compiled
				return compiled.MatchString(fullPath) || compiled.MatchString(baseName)
			}
		}
	}

	return false
}

func (pm *PatternMatcher) isRegexPattern(pattern string) bool {
	// Simple heuristic to detect regex patterns
	regexChars := []string{"^", "$", "[", "]", "{", "}", "(", ")", "|", "+", "?", "\\"}
	for _, char := range regexChars {
		if strings.Contains(pattern, char) {
			return true
		}
	}
	return false
}

func (pm *PatternMatcher) AddPattern(pattern string, level RiskLevel) {
	pm.patterns[level] = append(pm.patterns[level], pattern)
}

func (pm *PatternMatcher) RemovePattern(pattern string, level RiskLevel) {
	patterns := pm.patterns[level]
	for i, p := range patterns {
		if p == pattern {
			pm.patterns[level] = append(patterns[:i], patterns[i+1:]...)
			break
		}
	}
}

func (pm *PatternMatcher) GetPatterns(level RiskLevel) []string {
	return append([]string{}, pm.patterns[level]...) // Return copy
}

func (pm *PatternMatcher) GetAllPatterns() map[RiskLevel][]string {
	result := make(map[RiskLevel][]string)
	for level, patterns := range pm.patterns {
		result[level] = append([]string{}, patterns...) // Return copies
	}
	return result
}
