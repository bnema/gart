package security

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

const (
	// Entropy thresholds for different sensitivity levels
	LowEntropyThreshold    = 3.5
	MediumEntropyThreshold = 4.5
	HighEntropyThreshold   = 5.5

	// Minimum length for entropy-based detection
	MinEntropyLength = 20

	// Maximum length to consider (avoid very long strings like base64 images)
	MaxEntropyLength = 200
)

type EntropyAnalyzer struct {
	threshold float64
}

func NewEntropyAnalyzer(threshold float64) *EntropyAnalyzer {
	return &EntropyAnalyzer{
		threshold: threshold,
	}
}

// CalculateEntropy calculates Shannon entropy for a string
func CalculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count character frequencies
	frequency := make(map[rune]int)
	for _, char := range s {
		frequency[char]++
	}

	// Calculate entropy
	var entropy float64
	length := float64(len(s))

	for _, count := range frequency {
		if count > 0 {
			probability := float64(count) / length
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

// IsHighEntropy checks if a string has high entropy indicating randomness
func (ea *EntropyAnalyzer) IsHighEntropy(s string) bool {
	if len(s) < MinEntropyLength {
		return false
	}

	// Skip very long strings (likely base64 images, etc.)
	if len(s) > MaxEntropyLength {
		return false
	}

	// Check for known secret prefixes which should be flagged regardless of entropy
	lower := strings.ToLower(s)
	secretPrefixes := []string{"sk-", "pk-", "api-", "key-", "tok-", "akia", "eyj"}

	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lower, prefix) && len(s) >= MinEntropyLength {
			// For known secret patterns, use a lower threshold
			entropy := CalculateEntropy(s)
			return entropy >= ea.threshold*0.8 // 20% more lenient for known patterns
		}
	}

	entropy := CalculateEntropy(s)
	return entropy >= ea.threshold
}

// AnalyzeString provides detailed entropy analysis
func (ea *EntropyAnalyzer) AnalyzeString(s string) EntropyAnalysis {
	analysis := EntropyAnalysis{
		Value:      s,
		Length:     len(s),
		Entropy:    CalculateEntropy(s),
		IsSecret:   false,
		Confidence: 0.0,
	}

	if analysis.Length < MinEntropyLength {
		analysis.Reason = "String too short for entropy analysis"
		return analysis
	}

	if analysis.Length > MaxEntropyLength {
		analysis.Reason = "String too long, likely not a secret"
		return analysis
	}

	// Calculate confidence based on entropy and other factors
	analysis.Confidence = ea.calculateConfidence(s, analysis.Entropy)
	analysis.IsSecret = analysis.Confidence > 0.7

	if analysis.IsSecret {
		analysis.Reason = "High entropy indicates randomness typical of secrets"
	} else {
		analysis.Reason = "Entropy too low for secret detection"
	}

	return analysis
}

type EntropyAnalysis struct {
	Value      string
	Length     int
	Entropy    float64
	IsSecret   bool
	Confidence float64
	Reason     string
}

func (ea *EntropyAnalyzer) calculateConfidence(s string, entropy float64) float64 {
	// Early exit for structural patterns
	if isStructured, conf := ea.analyzeStructuralPattern(s); isStructured {
		return conf // Very low confidence for UI/command patterns
	}
	
	// Check natural language before entropy
	nlScore := ea.calculateNaturalLanguageScore(s)
	if nlScore > 0.6 {
		// High natural language content caps confidence
		return math.Min(0.3, entropy / 10.0)
	}

	// Start with entropy-based confidence
	var baseConfidence float64
	if entropy >= HighEntropyThreshold {
		baseConfidence = 0.9
	} else if entropy >= MediumEntropyThreshold {
		baseConfidence = 0.75
	} else if entropy >= LowEntropyThreshold {
		baseConfidence = 0.5
	} else if entropy >= 3.0 {
		baseConfidence = 0.3
	} else {
		return 0.0
	}

	// Get individual scores
	compositionScore := ea.analyzeCharacterComposition(s)
	lengthScore := ea.analyzeLengthPattern(s)
	patternScore := ea.filterCommonPatterns(s)

	// Natural language heavily reduces confidence
	if nlScore > 0.3 {
		patternScore *= (1.0 - nlScore*0.5)
	}

	// Calculate weighted average with pattern score having veto power
	// If pattern score is very low (< 0.3), it's likely a placeholder
	if patternScore < 0.3 {
		return patternScore * baseConfidence
	}

	// Otherwise, use a weighted average that doesn't overly diminish confidence
	confidence := (baseConfidence * 0.5) + (compositionScore * 0.25) + (lengthScore * 0.25)

	// Apply pattern score as a multiplier, but cap the reduction
	confidence *= math.Max(patternScore, 0.7)

	// Boost for very high entropy strings
	if entropy > 5.0 && len(s) > 30 {
		confidence = math.Max(confidence, 0.8)
	}

	return math.Min(confidence, 1.0)
}

func (ea *EntropyAnalyzer) analyzeCharacterComposition(s string) float64 {
	var (
		hasUpper     bool
		hasLower     bool
		hasDigit     bool
		hasSpecial   bool
		letterCount  int
		digitCount   int
		specialCount int
	)

	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
			letterCount++
		case unicode.IsLower(char):
			hasLower = true
			letterCount++
		case unicode.IsDigit(char):
			hasDigit = true
			digitCount++
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
			specialCount++
		}
	}

	score := 0.5 // Base score

	// Mixed case is good for secrets
	if hasUpper && hasLower {
		score += 0.2
	}

	// Mix of letters and numbers is typical for API keys
	if hasDigit && letterCount > 0 {
		score += 0.3
	}

	// Some special characters are common in secrets (but not too many)
	if hasSpecial && specialCount <= len(s)/4 {
		score += 0.1
	}

	// Check ratios
	if len(s) > 0 {
		totalChars := len(s)
		letterRatio := float64(letterCount) / float64(totalChars)
		digitRatio := float64(digitCount) / float64(totalChars)

		// Good balance of letters and numbers
		if letterRatio > 0.3 && letterRatio < 0.9 && digitRatio > 0.1 && digitRatio < 0.7 {
			score += 0.2
		}

		// Prefer mixed character types
		charTypes := 0
		if hasUpper || hasLower {
			charTypes++
		}
		if hasDigit {
			charTypes++
		}
		if hasSpecial {
			charTypes++
		}

		if charTypes >= 2 {
			score += 0.1
		}
	}

	return math.Min(score, 1.0)
}

func (ea *EntropyAnalyzer) analyzeLengthPattern(s string) float64 {
	length := len(s)

	// Normalize scores between 0 and 1
	switch {
	case length < MinEntropyLength: // Too short
		return 0.2
	case length == 32: // MD5 hash length
		return 0.9
	case length == 40: // SHA1 hash length
		return 0.9
	case length == 64: // SHA256 hash length
		return 0.85
	case length >= 20 && length <= 50: // Common API key range
		return 1.0 // Perfect length for secrets
	case length >= 50 && length <= 100: // Longer tokens
		return 0.9
	case length > 100 && length <= MaxEntropyLength: // Very long, might be JWT or certificate
		return 0.7
	case length > MaxEntropyLength: // Too long
		return 0.3
	default:
		return 0.6
	}
}

// analyzeStructuralPattern detects UI text, commands, and other structured patterns
func (ea *EntropyAnalyzer) analyzeStructuralPattern(s string) (isStructured bool, confidence float64) {
	// UI Mnemonic pattern: [X] or [X]text where X is 1-2 chars
	uiMnemonicPattern := regexp.MustCompile(`^\[[A-Za-z ]{1,2}\]`)
	if uiMnemonicPattern.MatchString(s) {
		// Extract text after bracket
		afterBracket := uiMnemonicPattern.ReplaceAllString(s, "")
		// If remaining text is mostly alphabetic with spaces, it's UI text
		alphaSpaceCount := 0
		for _, r := range afterBracket {
			if unicode.IsLetter(r) || unicode.IsSpace(r) {
				alphaSpaceCount++
			}
		}
		if len(afterBracket) > 0 && float64(alphaSpaceCount)/float64(len(afterBracket)) > 0.8 {
			return true, 0.1
		}
	}
	
	// Vim command pattern: :CommandName<Key>
	vimCmdPattern := regexp.MustCompile(`^:[A-Za-z][A-Za-z0-9]*(<[A-Z]+>)?$`)
	if vimCmdPattern.MatchString(s) {
		return true, 0.1
	}
	
	// Checkbox pattern: [ ] text or [x] text
	checkboxPattern := regexp.MustCompile(`^\[[x \-]\]\s+.+`)
	if checkboxPattern.MatchString(s) {
		return true, 0.1
	}
	
	return false, 1.0
}

// calculateNaturalLanguageScore determines how much of the string is natural language
func (ea *EntropyAnalyzer) calculateNaturalLanguageScore(s string) float64 {
	words := strings.Fields(s)
	if len(words) == 0 {
		return 0.0
	}
	
	naturalWordCount := 0
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true,
		"find": true, "search": true, "file": true, "buffer": true, "existing": true,
		"toggle": true, "tree": true, "nvim": true, "vim": true, "goto": true,
		"help": true, "tags": true, "recent": true, "files": true, "current": true,
		"document": true, "symbols": true, "references": true, "definition": true,
		"implementation": true, "type": true, "workspace": true, "dynamic": true,
	}
	
	for _, word := range words {
		cleanWord := strings.Trim(strings.ToLower(word), "[]()<>:;,.")
		if len(cleanWord) >= 2 && (commonWords[cleanWord] || ea.isCommonEnglishWord(cleanWord)) {
			naturalWordCount++
		}
	}
	
	return float64(naturalWordCount) / float64(len(words))
}

// isCommonEnglishWord checks if a word is likely common English
func (ea *EntropyAnalyzer) isCommonEnglishWord(word string) bool {
	// Check if word is mostly alphabetic
	alphaCount := 0
	for _, r := range word {
		if unicode.IsLetter(r) {
			alphaCount++
		}
	}
	// Word should be 2+ chars, mostly letters, and not look like hex/base64
	return len(word) >= 2 && 
		   float64(alphaCount)/float64(len(word)) > 0.8 &&
		   !regexp.MustCompile(`^[a-f0-9]+$`).MatchString(word)
}

func (ea *EntropyAnalyzer) filterCommonPatterns(s string) float64 {
	lower := strings.ToLower(s)
	
	// Check structural patterns first (highest priority)
	if isStructured, conf := ea.analyzeStructuralPattern(s); isStructured {
		return conf
	}
	
	// Check natural language score
	nlScore := ea.calculateNaturalLanguageScore(s)
	if nlScore > 0.6 {
		// Mostly natural language, very unlikely to be a secret
		return 0.2
	}

	// Check for GitHub/package patterns
	if matched := regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`).MatchString(s); matched {
		return 0.2 // Very low confidence for package names
	}

	// Check URLs - separate domain from path
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		// Parse URL to check if it contains credentials
		if regexp.MustCompile(`https?://[^:]+:[^@]+@`).MatchString(lower) {
			return 1.0 // URL with credentials, likely secret
		}
		// Check for API endpoints that are just base URLs
		if regexp.MustCompile(`^https?://[a-z0-9.-]+\.(com|org|net|io)/?$`).MatchString(lower) {
			return 0.1 // Just a domain, not a secret
		}
		// Check for known public APIs
		publicAPIs := []string{"github.com", "gitlab.com", "npmjs.org", "pypi.org", 
							   "api.anthropic.com", "api.openai.com", "googleapis.com"}
		for _, api := range publicAPIs {
			if strings.Contains(lower, api) {
				return 0.15 // Known public API endpoint
			}
		}
	}

	// Check for known secret prefixes (these are likely real secrets)
	secretPrefixes := []string{
		"sk-", "pk-", "api-", "key-", "tok-", "akia", "eyj", // Common API key/token prefixes
		"sk-ant-api03-", // Anthropic keys
	}

	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return 1.2 // Boost confidence for known secret patterns
		}
	}

	// Enhanced config pattern detection with word boundaries
	// Only flag if it's NOT a natural language context
	if nlScore < 0.4 {
		commonConfigPatterns := []string{
			// Package/namespace identifiers
			"com.", "org.", "net.", "io.", "co.", "de.", "fr.", "uk.",
			// System paths
			"/usr/", "/var/", "/etc/", "/home/", "/opt/", "/tmp/", "/bin/", "/sbin/",
			"/.config/", "/.local/", "/.cache/", "~/.config/", "~/.local/",
			"c:\\", "program files", "appdata", "documents",
			// Color and theme related
			"color", "rgb", "rgba", "hex", "#", "bold", "italic", "underline", "normal",
			"bright", "dim", "foreground", "background", "fg", "bg",
			// Shell/terminal patterns
			"bash", "zsh", "fish", "sh", "cmd", "powershell", "terminal",
			"console", "completion", "function", "alias", "export",
			// File extensions
			".fish", ".sh", ".py", ".js", ".json", ".xml", ".yaml", ".toml",
			// Common words that shouldn't be secrets
			"settings", "config", "options", "preferences", "profile",
			"theme", "layout", "display", "format", "style", "appearance",
			"favorites", "bookmarks", "history", "recent", "cache",
			"server", "client", "host", "port", "address", "connection",
			"substitution", "environment", "downloading", "available",
			// Unix socket paths
			"unix:", ".sock",
		}

		for _, pattern := range commonConfigPatterns {
			if strings.Contains(lower, pattern) {
				return 0.3 // Low confidence for config patterns
			}
		}
	}

	// Check for common color names
	colorNames := []string{
		"red", "green", "blue", "yellow", "cyan", "magenta", "black", "white",
		"gray", "grey", "purple", "orange", "pink", "brown", "violet", "indigo",
		"navy", "teal", "lime", "olive", "maroon", "silver", "gold",
	}

	for _, color := range colorNames {
		if strings.Contains(lower, color) {
			return 0.2 // Very low confidence for color-related strings
		}
	}

	// Check for natural language phrases
	naturalPhrases := []string{
		"don't", "can't", "won't", "isn't", "aren't", "wasn't", "weren't",
		"haven't", "hasn't", "hadn't", "wouldn't", "couldn't", "shouldn't",
		"anything", "something", "nothing", "everything", "somewhere",
		"anywhere", "nowhere", "everywhere", "someone", "anyone",
		"everyone", "nobody", "everybody", "write", "read", "available",
		"downloading", "processing", "substitution", "replacement",
	}

	for _, phrase := range naturalPhrases {
		if strings.Contains(lower, phrase) {
			return 0.1 // Very low confidence for natural language
		}
	}

	// Whole-string patterns that indicate non-secrets
	if lower == "example" || lower == "test" || lower == "demo" || lower == "sample" ||
		lower == "placeholder" || lower == "dummy" || lower == "fake" || lower == "mock" {
		return 0.1 // Very low confidence for obvious placeholders
	}

	// For longer strings, check if they're mostly placeholder patterns
	placeholderKeywords := []string{
		"example", "test", "demo", "sample", "placeholder", "dummy", "fake", "mock", "lorem", "ipsum",
	}

	placeholderCount := 0
	for _, keyword := range placeholderKeywords {
		if strings.Contains(lower, keyword) {
			placeholderCount++
		}
	}

	// Only penalize if multiple placeholder keywords or they make up a significant portion
	if placeholderCount >= 2 || (placeholderCount == 1 && len(s) < 20) {
		return 0.3
	}

	// Special case: if string starts with a placeholder keyword followed by underscore
	for _, keyword := range placeholderKeywords {
		if strings.HasPrefix(lower, keyword+"_") || strings.HasSuffix(lower, "_"+keyword) {
			return 0.2 // Very likely a placeholder
		}
	}

	// Don't penalize for common sequences like "123456" or "abcdef" if they're part of a larger high-entropy string
	if len(s) > 20 && CalculateEntropy(s) > 4.0 {
		// High entropy long strings are likely legitimate even with common subsequences
		return 1.0
	}

	// Check for repeated patterns
	if ea.hasRepeatedPattern(s) {
		return 0.5
	}

	// Check for common hash-like patterns that aren't secrets
	if ea.isLikelyHash(s) {
		return 0.8 // Hashes might still be sensitive
	}

	return 1.0
}

func (ea *EntropyAnalyzer) hasRepeatedPattern(s string) bool {
	// Look for repeated substrings
	for patternLen := 2; patternLen <= len(s)/3; patternLen++ {
		pattern := s[:patternLen]
		repetitions := strings.Count(s, pattern)
		if repetitions >= 3 {
			return true
		}
	}
	return false
}

func (ea *EntropyAnalyzer) isLikelyHash(s string) bool {
	// Check if string looks like a hash (all hex characters)
	if len(s) == 32 || len(s) == 40 || len(s) == 64 {
		for _, char := range s {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
				return false
			}
		}
		return true
	}
	return false
}

// ExtractHighEntropyStrings finds all high-entropy strings in text
func (ea *EntropyAnalyzer) ExtractHighEntropyStrings(text string) []string {
	var results []string

	// Split by common delimiters
	delimiters := []string{" ", "\t", "\n", "\r", "\"", "'", "=", ":", ",", ";", "(", ")", "[", "]", "{", "}", "<", ">"}

	tokens := []string{text}
	for _, delimiter := range delimiters {
		var newTokens []string
		for _, token := range tokens {
			parts := strings.Split(token, delimiter)
			newTokens = append(newTokens, parts...)
		}
		tokens = newTokens
	}

	// Check each token
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if ea.IsHighEntropy(token) {
			results = append(results, token)
		}
	}

	return results
}
