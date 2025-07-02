package security

import (
	"fmt"
	"regexp"
	"strings"
)

type Detector struct {
	config          *SecurityConfig
	entropyAnalyzer *EntropyAnalyzer
	patterns        map[SecretType]*regexp.Regexp
	contextKeywords []string
}

type SensitivityLevel string

const (
	SensitivityLow      SensitivityLevel = "low"
	SensitivityMedium   SensitivityLevel = "medium"
	SensitivityHigh     SensitivityLevel = "high"
	SensitivityParanoid SensitivityLevel = "paranoid"
)

func NewDetector(config *SecurityConfig) *Detector {
	threshold := MediumEntropyThreshold
	if config != nil {
		threshold = config.ContentScan.EntropyThreshold
	}

	detector := &Detector{
		config:          config,
		entropyAnalyzer: NewEntropyAnalyzer(threshold),
		patterns:        make(map[SecretType]*regexp.Regexp),
		contextKeywords: getContextKeywords(),
	}

	detector.compilePatterns()
	return detector
}

func (d *Detector) compilePatterns() {
	secretPatterns := map[SecretType]string{
		// OpenAI API Keys (various formats)
		SecretTypeAPIKey: `(sk-[a-zA-Z0-9]{20}T3BlbkFJ[a-zA-Z0-9]{20}|sk-proj-[a-zA-Z0-9_-]{43}T3BlbkFJ[a-zA-Z0-9_-]{20}|sk-[a-zA-Z0-9]{48})`,

		// Anthropic API Keys
		SecretTypeAnthropicKey: `(sk-ant-api03-[a-zA-Z0-9_-]{95})`,

		// Generic API Keys (including environment variables ending in _API_KEY)
		SecretTypeGenericAPIKey: `(?i)([a-zA-Z0-9_]*api[_-]?key[s]?['"]*\s*[=:]\s*['"]*([a-zA-Z0-9_-]{20,})|sk-[a-zA-Z0-9_-]{20,})`,

		// AWS
		SecretTypeAWSKey: `(AKIA[0-9A-Z]{16}|aws_access_key_id|aws_secret_access_key)`,

		// GitHub
		SecretTypeGitHubToken: `(gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[a-zA-Z0-9_]{82})`,

		// JWT
		SecretTypeJWT: `(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`,

		// Private Keys
		SecretTypePrivateKey: `-----BEGIN\s+(?:RSA\s+|DSA\s+|EC\s+|OPENSSH\s+)?PRIVATE\s+KEY-----`,

		// Passwords
		SecretTypePassword: `(?i)password[s]?['"]*\s*[=:]\s*['"]*([^\s'"]{8,})`,

		// Database URLs
		SecretTypeDatabaseURL: `(postgres|mysql|mongodb)://[^:]+:[^@]+@[^/]+`,

		// Generic tokens
		SecretTypeToken: `(?i)token[s]?['"]*\s*[=:]\s*['"]*([a-zA-Z0-9_-]{20,})`,

		// Email addresses
		SecretTypeEmail: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,

		// IP Addresses (private ranges)
		SecretTypeIPAddress: `\b(?:10\.|172\.(?:1[6-9]|2[0-9]|3[01])\.|192\.168\.)\d{1,3}\.\d{1,3}\b`,

		// Credit Card Numbers (simple pattern)
		SecretTypeCreditCard: `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3[0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`,

		// Social Security Numbers
		SecretTypeSSN: `\b\d{3}-\d{2}-\d{4}\b`,
	}

	for secretType, pattern := range secretPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			d.patterns[secretType] = compiled
		}
	}
}

func getContextKeywords() []string {
	return []string{
		"key", "token", "secret", "password", "pass", "pwd", "auth", "credential",
		"api", "oauth", "jwt", "bearer", "access", "private", "public", "cert",
		"certificate", "signature", "hash", "salt", "encrypt", "decrypt", "cipher",
		"login", "username", "user", "admin", "root", "database", "db", "connection",
		"url", "endpoint", "host", "server", "client", "config", "setting", "env",
		"environment", "dev", "prod", "production", "staging", "test", "demo",
	}
}

func (d *Detector) DetectSecrets(content []byte, filePath string) []Finding {
	var findings []Finding

	text := string(content)
	lines := strings.Split(text, "\n")

	// Pattern-based detection
	for secretType, pattern := range d.patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 0 {
				finding := d.createFinding(secretType, match[0], text, filePath)
				if finding != nil {
					findings = append(findings, *finding)
				}
			}
		}
	}

	// Entropy-based detection
	for lineNum, line := range lines {
		entropyFindings := d.detectEntropySecrets(line, lineNum+1, filePath)
		findings = append(findings, entropyFindings...)
	}

	// Context-aware detection
	contextFindings := d.detectContextSecrets(lines, filePath)
	findings = append(findings, contextFindings...)

	// Remove duplicates and merge similar findings
	findings = d.deduplicateFindings(findings)

	return findings
}

func (d *Detector) createFinding(secretType SecretType, value, text, filePath string) *Finding {
	// Skip if value is too short or looks like a placeholder
	if len(value) < 8 || d.isPlaceholder(value) {
		return nil
	}

	// Find line number and context
	lines := strings.Split(text, "\n")
	lineNum := 1
	lineText := ""

	for i, line := range lines {
		if strings.Contains(line, value) {
			lineNum = i + 1
			lineText = line
			break
		}
	}

	// Calculate confidence
	confidence := d.calculatePatternConfidence(secretType, value, lineText)

	finding := &Finding{
		Type:       secretType,
		RawValue:   value,
		Value:      redactValue(value),
		Location:   Location{FilePath: filePath, LineNumber: lineNum, LineText: lineText},
		Confidence: confidence,
		Context:    d.extractContext(text, value),
		Reasons:    []string{string(secretType) + " pattern match"},
		RiskLevel:  d.getRiskLevel(secretType, confidence),
	}

	return finding
}

func (d *Detector) detectEntropySecrets(line string, lineNum int, filePath string) []Finding {
	var findings []Finding

	// Extract potential secrets from the line
	candidates := d.extractSecretCandidates(line)

	for _, candidate := range candidates {
		analysis := d.entropyAnalyzer.AnalyzeString(candidate)
		if analysis.IsSecret {
			// Check if this looks like a legitimate secret
			if d.isLikelySecret(candidate, line) {
				finding := Finding{
					Type:       SecretTypeGeneric,
					RawValue:   candidate,
					Value:      redactValue(candidate),
					Location:   Location{FilePath: filePath, LineNumber: lineNum, LineText: line},
					Confidence: analysis.Confidence,
					Context:    line,
					Reasons:    []string{"High entropy string", analysis.Reason},
					RiskLevel:  d.getRiskLevel(SecretTypeGeneric, analysis.Confidence),
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

func (d *Detector) detectContextSecrets(lines []string, filePath string) []Finding {
	var findings []Finding

	for lineNum, line := range lines {
		// Look for key-value patterns with sensitive keywords
		for _, keyword := range d.contextKeywords {
			if d.containsKeyword(line, keyword) {
				values := d.extractValuesNearKeyword(line, keyword)
				for _, value := range values {
					if len(value) >= 8 && !d.isPlaceholder(value) {
						confidence := d.calculateContextConfidence(keyword, value, line)
						if confidence > 0.5 {
							finding := Finding{
								Type:       d.inferSecretType(keyword, value),
								RawValue:   value,
								Value:      redactValue(value),
								Location:   Location{FilePath: filePath, LineNumber: lineNum + 1, LineText: line},
								Confidence: confidence,
								Context:    line,
								Reasons:    []string{"Found near sensitive keyword: " + keyword},
								RiskLevel:  d.getRiskLevel(SecretTypeGeneric, confidence),
							}
							findings = append(findings, finding)
						}
					}
				}
			}
		}
	}

	return findings
}

func (d *Detector) extractSecretCandidates(line string) []string {
	var candidates []string

	// Extract quoted strings
	quotedStrings := extractQuotedStrings(line)
	candidates = append(candidates, quotedStrings...)

	// Extract values after = or :
	valuePatterns := []*regexp.Regexp{
		regexp.MustCompile(`[=:]\s*([^\s"',;]+)`),
		regexp.MustCompile(`[=:]\s*"([^"]+)"`),
		regexp.MustCompile(`[=:]\s*'([^']+)'`),
	}

	for _, pattern := range valuePatterns {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				candidates = append(candidates, match[1])
			}
		}
	}

	return candidates
}

func (d *Detector) isPlaceholder(value string) bool {
	lower := strings.ToLower(value)
	placeholders := []string{
		"example", "test", "demo", "sample", "placeholder", "dummy", "fake",
		"mock", "lorem", "ipsum", "your_", "insert_", "replace_", "change_",
		"xxxx", "****", "....", "null", "none", "empty", "todo", "fixme",
	}

	for _, placeholder := range placeholders {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}

	// Check for repeated characters, but not for potential credit cards
	if d.mightBeCreditCard(value) {
		return false
	}

	if d.hasRepeatedChar(value) {
		return true
	}

	return false
}

func (d *Detector) mightBeCreditCard(value string) bool {
	// Credit cards are typically 13-19 digits
	if len(value) < 13 || len(value) > 19 {
		return false
	}

	// Must be all digits
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Common credit card prefixes
	return strings.HasPrefix(value, "4") || // Visa
		strings.HasPrefix(value, "5") || // Mastercard
		strings.HasPrefix(value, "3") || // American Express/Diners
		strings.HasPrefix(value, "6") // Discover
}

func (d *Detector) hasRepeatedChar(s string) bool {
	if len(s) < 4 {
		return false
	}

	charCount := make(map[rune]int)
	for _, char := range s {
		charCount[char]++
	}

	for _, count := range charCount {
		if count > len(s)/2 {
			return true
		}
	}

	return false
}

func (d *Detector) isLikelySecret(candidate, context string) bool {
	// Additional heuristics to determine if a high-entropy string is likely a secret

	// Check for common non-secret patterns FIRST
	if d.isCommonConfigPattern(candidate, context) {
		return false
	}

	// Too short
	if len(candidate) < 16 {
		return false
	}

	// All same character
	if d.hasRepeatedChar(candidate) {
		return false
	}

	// Contains dictionary words (less likely to be a secret)
	if d.containsDictionaryWords(candidate) {
		return false
	}

	// Check context for secret-related keywords
	contextScore := 0
	lowerContext := strings.ToLower(context)
	for _, keyword := range d.contextKeywords {
		if strings.Contains(lowerContext, keyword) {
			contextScore++
		}
	}

	// If we find secret-related keywords, it's likely a secret
	if contextScore > 0 {
		return true
	}

	// For high entropy strings without clear secret context,
	// apply additional checks based on string characteristics
	return d.hasSecretCharacteristics(candidate)
}

func (d *Detector) containsDictionaryWords(s string) bool {
	// Expanded check for common English words and patterns
	commonWords := []string{
		"the", "and", "for", "are", "but", "not", "you", "all", "can", "had",
		"her", "was", "one", "our", "out", "day", "get", "has", "him", "his",
		"how", "man", "new", "now", "old", "see", "two", "way", "who", "boy",
		"did", "its", "let", "put", "say", "she", "too", "use", "don", "any",
		"thing", "about", "would", "there", "could", "other", "after", "first",
		"well", "over", "think", "also", "your", "work", "life", "only", "back",
		"even", "good", "woman", "through", "down", "way", "look", "right",
		"system", "computer", "program", "government", "company", "group",
		"part", "place", "case", "point", "hand", "high", "important", "public",
		"number", "fact", "be", "have", "do", "say", "go", "can", "get", "or",
		"will", "my", "one", "all", "would", "there", "their", "what", "so",
		"up", "out", "if", "about", "who", "get", "which", "go", "when",
		"make", "can", "like", "time", "no", "just", "him", "know", "take",
		"people", "into", "year", "your", "good", "some", "could", "them",
		"see", "other", "than", "then", "now", "look", "only", "come", "its",
		"over", "think", "also", "back", "after", "use", "two", "how", "our",
		"work", "first", "well", "way", "even", "new", "want", "because",
		"any", "these", "give", "day", "most", "us", "is", "water", "long",
		"very", "what", "know", "through", "back", "much", "before", "move",
		"right", "boy", "old", "too", "same", "tell", "does", "set", "three",
		"want", "air", "well", "also", "play", "small", "end", "put", "home",
		"read", "hand", "port", "large", "spell", "add", "even", "land", "here",
		"must", "big", "high", "such", "follow", "act", "why", "ask", "men",
		"change", "went", "light", "kind", "off", "need", "house", "picture",
		"try", "again", "animal", "point", "mother", "world", "near", "build",
		"self", "earth", "father", "head", "stand", "own", "page", "should",
		"country", "found", "answer", "school", "grow", "study", "still",
		"learn", "plant", "cover", "food", "sun", "four", "between", "state",
		"keep", "eye", "never", "last", "let", "thought", "city", "tree",
		"cross", "farm", "hard", "start", "might", "story", "saw", "far",
		"sea", "draw", "left", "late", "run", "don't", "while", "press",
		"close", "night", "real", "life", "few", "north", "open", "seem",
		"together", "next", "white", "children", "begin", "got", "walk",
		"example", "ease", "paper", "group", "always", "music", "those",
		"both", "mark", "often", "letter", "until", "mile", "river", "car",
		"feet", "care", "second", "book", "carry", "took", "science", "eat",
		"room", "friend", "began", "idea", "fish", "mountain", "stop",
		"once", "base", "hear", "horse", "cut", "sure", "watch", "color",
		"face", "wood", "main", "enough", "plain", "girl", "usual", "young",
		"ready", "above", "ever", "red", "list", "though", "feel", "talk",
		"bird", "soon", "body", "dog", "family", "direct", "pose", "leave",
		"song", "measure", "door", "product", "black", "short", "numeral",
		"class", "wind", "question", "happen", "complete", "ship", "area",
		"half", "rock", "order", "fire", "south", "problem", "piece", "told",
		"knew", "pass", "since", "top", "whole", "king", "space", "heard",
		"best", "hour", "better", "during", "hundred", "five", "remember",
		"step", "early", "hold", "west", "ground", "interest", "reach",
		"fast", "verb", "sing", "listen", "six", "table", "travel", "less",
		"morning", "ten", "simple", "several", "vowel", "toward", "war",
		"lay", "against", "pattern", "slow", "center", "love", "person",
		"money", "serve", "appear", "road", "map", "rain", "rule", "govern",
		"pull", "cold", "notice", "voice", "unit", "power", "town", "fine",
		"certain", "fly", "fall", "lead", "cry", "dark", "machine", "note",
		"wait", "plan", "figure", "star", "box", "noun", "field", "rest",
		"correct", "able", "pound", "done", "beauty", "drive", "stood",
		"contain", "front", "teach", "week", "final", "gave", "green",
		"oh", "quick", "develop", "ocean", "warm", "free", "minute",
		"strong", "special", "mind", "behind", "clear", "tail", "produce",
		"fact", "street", "inch", "multiply", "nothing", "course", "stay",
		"wheel", "full", "force", "blue", "object", "decide", "surface",
		"deep", "moon", "island", "foot", "system", "busy", "test", "record",
		"boat", "common", "gold", "possible", "plane", "stead", "dry",
		"wonder", "laugh", "thousands", "ago", "ran", "check", "game",
		"shape", "equate", "hot", "miss", "brought", "heat", "snow",
		"tire", "bring", "yes", "distant", "fill", "east", "paint", "language",
		"among", "grand", "ball", "yet", "wave", "drop", "heart", "am",
		"present", "heavy", "dance", "engine", "position", "arm", "wide",
		"sail", "material", "size", "vary", "settle", "speak", "weight",
		"general", "ice", "matter", "circle", "pair", "include", "divide",
		"syllable", "felt", "perhaps", "pick", "sudden", "count", "square",
		"reason", "length", "represent", "art", "subject", "region", "energy",
		"hunt", "probable", "bed", "brother", "egg", "ride", "cell", "believe",
		"fraction", "forest", "sit", "race", "window", "store", "summer",
		"train", "sleep", "prove", "lone", "leg", "exercise", "wall", "catch",
		"mount", "wish", "sky", "board", "joy", "winter", "sat", "written",
		"wild", "instrument", "kept", "glass", "grass", "cow", "job", "edge",
		"sign", "visit", "past", "soft", "fun", "bright", "gas", "weather",
		"month", "million", "bear", "finish", "happy", "hope", "flower",
		"clothe", "strange", "gone", "jump", "baby", "eight", "village",
		"meet", "root", "buy", "raise", "solve", "metal", "whether", "push",
		"seven", "paragraph", "third", "shall", "held", "hair", "describe",
		"cook", "floor", "either", "result", "burn", "hill", "safe", "cat",
		"century", "consider", "type", "law", "bit", "coast", "copy", "phrase",
		"silent", "tall", "sand", "soil", "roll", "temperature", "finger",
		"industry", "value", "fight", "lie", "beat", "excite", "natural",
		"view", "sense", "ear", "else", "quite", "broke", "case", "middle",
		"kill", "son", "lake", "moment", "scale", "loud", "spring", "observe",
		"child", "straight", "consonant", "nation", "dictionary", "milk",
		"speed", "method", "organ", "pay", "age", "section", "dress", "cloud",
		"surprise", "quiet", "stone", "tiny", "climb", "bad", "oil", "blood",
		"touch", "grew", "cent", "mix", "team", "wire", "cost", "lost", "brown",
		"wear", "garden", "equal", "sent", "choose", "fell", "fit", "flow",
		"fair", "bank", "collect", "save", "control", "decimal", "gentle",
		"woman", "captain", "practice", "separate", "difficult", "doctor",
		"please", "protect", "noon", "whose", "locate", "ring", "character",
		"insect", "caught", "period", "indicate", "radio", "spoke", "atom",
		"human", "history", "effect", "electric", "expect", "crop", "modern",
		"element", "hit", "student", "corner", "party", "supply", "bone",
		"rail", "imagine", "provide", "agree", "thus", "capital", "won't",
		"chair", "danger", "fruit", "rich", "thick", "soldier", "process",
		"operate", "guess", "necessary", "sharp", "wing", "create", "neighbor",
		"wash", "bat", "rather", "crowd", "corn", "compare", "poem", "string",
		"bell", "depend", "meat", "rub", "tube", "famous", "dollar", "stream",
		"fear", "sight", "thin", "triangle", "planet", "hurry", "chief",
		"colony", "clock", "mine", "tie", "enter", "major", "fresh", "search",
		"send", "yellow", "gun", "allow", "print", "dead", "spot", "desert",
		"suit", "current", "lift", "rose", "continue", "block", "chart",
		"hat", "sell", "success", "company", "subtract", "event", "particular",
		"deal", "swim", "term", "opposite", "wife", "shoe", "shoulder",
		"spread", "arrange", "camp", "invent", "cotton", "born", "determine",
		"quart", "nine", "truck", "noise", "level", "chance", "gather",
		"shop", "stretch", "throw", "shine", "property", "column", "molecule",
		"select", "wrong", "gray", "repeat", "require", "broad", "prepare",
		"salt", "nose", "plural", "anger", "claim", "continent", "oxygen",
		"sugar", "death", "pretty", "skill", "women", "season", "solution",
		"magnet", "silver", "thank", "branch", "match", "suffix", "especially",
		"fig", "afraid", "huge", "sister", "steel", "discuss", "forward",
		"similar", "guide", "experience", "score", "apple", "bought", "led",
		"pitch", "coat", "mass", "card", "band", "rope", "slip", "win",
		"dream", "evening", "condition", "feed", "tool", "total", "basic",
		"smell", "valley", "nor", "double", "seat", "arrive", "master",
		"track", "parent", "shore", "division", "sheet", "substance", "favor",
		"connect", "post", "spend", "chord", "fat", "glad", "original",
		"share", "station", "dad", "bread", "charge", "proper", "bar",
		"offer", "segment", "slave", "duck", "instant", "market", "degree",
		"populate", "chick", "dear", "enemy", "reply", "drink", "occur",
		"support", "speech", "nature", "range", "steam", "motion", "path",
		"liquid", "log", "meant", "quotient", "teeth", "shell", "neck",
	}

	// Also check for common configuration patterns
	commonPatterns := []string{
		"config", "settings", "options", "preferences", "profile", "theme",
		"color", "background", "foreground", "bold", "italic", "underline",
		"font", "size", "family", "weight", "style", "decoration",
		"path", "directory", "folder", "file", "extension", "format",
		"server", "client", "host", "port", "address", "connection",
		"database", "table", "column", "row", "record", "field",
		"user", "username", "account", "profile", "session", "login",
		"application", "program", "software", "system", "service",
		"version", "release", "build", "revision", "branch", "commit",
		"environment", "development", "production", "staging", "testing",
		"framework", "library", "module", "package", "component",
		"interface", "protocol", "standard", "specification", "format",
		"encryption", "security", "authentication", "authorization",
		"performance", "optimization", "efficiency", "speed", "memory",
		"network", "internet", "web", "http", "https", "ftp", "ssh",
		"backup", "restore", "synchronization", "replication", "mirror",
		"monitoring", "logging", "debugging", "tracing", "profiling",
		"timeout", "interval", "delay", "duration", "frequency",
		"maximum", "minimum", "default", "custom", "automatic", "manual",
		"enabled", "disabled", "active", "inactive", "running", "stopped",
		"success", "failure", "error", "warning", "information", "debug",
		"input", "output", "source", "destination", "target", "reference",
		"template", "pattern", "regex", "expression", "formula", "algorithm",
		"function", "method", "procedure", "routine", "operation", "action",
		"event", "trigger", "handler", "callback", "listener", "observer",
		"filter", "search", "query", "request", "response", "result",
		"message", "notification", "alert", "reminder", "prompt", "dialog",
		"window", "panel", "tab", "menu", "toolbar", "statusbar", "sidebar",
		"button", "checkbox", "radio", "dropdown", "listbox", "textbox",
		"label", "caption", "title", "header", "footer", "content", "body",
		"layout", "design", "appearance", "visual", "graphic", "image",
		"icon", "symbol", "logo", "avatar", "thumbnail", "preview",
		"animation", "transition", "effect", "filter", "transformation",
		"encoding", "decoding", "compression", "decompression", "archive",
		"import", "export", "upload", "download", "transfer", "migration",
		"installation", "deployment", "configuration", "setup", "initialization",
		"maintenance", "update", "upgrade", "patch", "hotfix", "bugfix",
		"feature", "enhancement", "improvement", "optimization", "refactoring",
		"documentation", "manual", "guide", "tutorial", "example", "sample",
		"license", "copyright", "trademark", "patent", "legal", "compliance",
		"privacy", "policy", "terms", "conditions", "agreement", "contract",
		"support", "help", "assistance", "contact", "feedback", "report",
		"statistics", "analytics", "metrics", "measurement", "tracking",
		"cache", "buffer", "queue", "stack", "heap", "pool", "registry",
		"index", "catalog", "directory", "listing", "inventory", "repository",
		"workspace", "project", "solution", "portfolio", "collection",
		"category", "group", "class", "type", "kind", "sort", "order",
		"priority", "importance", "urgency", "severity", "level", "grade",
		"status", "state", "condition", "mode", "phase", "stage", "step",
		"process", "workflow", "pipeline", "chain", "sequence", "series",
		"cycle", "loop", "iteration", "repetition", "recursion", "nesting",
		"hierarchy", "structure", "organization", "arrangement", "composition",
		"relationship", "association", "connection", "link", "reference",
		"dependency", "requirement", "prerequisite", "constraint", "limitation",
		"exception", "error", "fault", "failure", "issue", "problem", "bug",
		"solution", "fix", "patch", "workaround", "alternative", "option",
		"choice", "selection", "decision", "determination", "resolution",
		"recommendation", "suggestion", "advice", "tip", "hint", "clue",
		"indication", "signal", "sign", "marker", "flag", "tag", "label",
		"identifier", "name", "title", "caption", "description", "comment",
		"note", "remark", "observation", "finding", "discovery", "insight",
		"knowledge", "information", "data", "facts", "details", "specifics",
		"general", "common", "typical", "standard", "normal", "regular",
		"special", "unique", "distinct", "different", "various", "multiple",
		"single", "individual", "separate", "independent", "standalone",
		"integrated", "combined", "merged", "unified", "consolidated",
		"distributed", "scattered", "spread", "widespread", "extensive",
		"comprehensive", "complete", "full", "entire", "whole", "total",
		"partial", "incomplete", "limited", "restricted", "constrained",
		"unlimited", "unrestricted", "free", "open", "public", "private",
		"internal", "external", "local", "remote", "global", "universal",
		"specific", "particular", "exact", "precise", "accurate", "correct",
		"approximate", "rough", "estimated", "calculated", "computed",
		"generated", "created", "produced", "manufactured", "constructed",
		"built", "developed", "designed", "planned", "intended", "proposed",
		"suggested", "recommended", "preferred", "selected", "chosen",
		"accepted", "approved", "confirmed", "verified", "validated",
		"tested", "checked", "examined", "inspected", "reviewed",
		"analyzed", "evaluated", "assessed", "measured", "compared",
		"contrasted", "distinguished", "differentiated", "separated",
		"isolated", "extracted", "derived", "obtained", "acquired",
		"received", "collected", "gathered", "assembled", "compiled",
		"organized", "structured", "formatted", "styled", "designed",
		"customized", "personalized", "tailored", "adapted", "modified",
		"adjusted", "tuned", "optimized", "improved", "enhanced",
		"upgraded", "updated", "refreshed", "renewed", "replaced",
		"substituted", "exchanged", "swapped", "switched", "changed",
		"transformed", "converted", "translated", "interpreted", "processed",
		"handled", "managed", "controlled", "operated", "executed",
		"performed", "accomplished", "achieved", "completed", "finished",
		"terminated", "ended", "stopped", "halted", "paused", "suspended",
		"resumed", "continued", "proceeded", "advanced", "progressed",
		"moved", "shifted", "transferred", "migrated", "relocated",
		"positioned", "placed", "located", "situated", "established",
		"installed", "deployed", "implemented", "applied", "utilized",
		"used", "employed", "engaged", "involved", "participated",
		"contributed", "provided", "supplied", "delivered", "offered",
		"presented", "displayed", "shown", "exhibited", "demonstrated",
		"illustrated", "explained", "described", "detailed", "specified",
		"defined", "outlined", "summarized", "highlighted", "emphasized",
		"stressed", "underlined", "marked", "noted", "mentioned",
		"referenced", "cited", "quoted", "included", "contained",
		"comprised", "consisted", "composed", "formed", "constituted",
		"represented", "symbolized", "indicated", "signified", "meant",
		"implied", "suggested", "hinted", "alluded", "referred",
		"pointed", "directed", "guided", "led", "conducted", "managed",
		"supervised", "monitored", "observed", "watched", "tracked",
		"followed", "traced", "recorded", "documented", "logged",
		"reported", "announced", "declared", "stated", "expressed",
		"communicated", "conveyed", "transmitted", "sent", "delivered",
		"distributed", "shared", "published", "released", "issued",
		"launched", "started", "initiated", "begun", "commenced",
		"triggered", "activated", "enabled", "turned", "switched",
		"powered", "energized", "charged", "loaded", "filled",
		"populated", "occupied", "used", "utilized", "employed",
		"applied", "implemented", "executed", "run", "operated",
		"controlled", "managed", "handled", "processed", "treated",
		"dealt", "addressed", "tackled", "approached", "handled",
		"resolved", "solved", "fixed", "repaired", "corrected",
		"adjusted", "modified", "changed", "altered", "updated",
		"revised", "improved", "enhanced", "optimized", "refined",
		"polished", "perfected", "finalized", "completed", "concluded",
	}

	lower := strings.ToLower(s)
	wordCount := 0
	patternCount := 0

	// Check for common English words
	for _, word := range commonWords {
		if strings.Contains(lower, word) {
			wordCount++
		}
	}

	// Check for common configuration patterns
	for _, pattern := range commonPatterns {
		if strings.Contains(lower, pattern) {
			patternCount++
		}
	}

	// Lower threshold - if we find any dictionary words or config patterns, it's likely not a secret
	return wordCount > 0 || patternCount > 0
}

func (d *Detector) containsKeyword(line, keyword string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, strings.ToLower(keyword))
}

func (d *Detector) extractValuesNearKeyword(line, keyword string) []string {
	var values []string

	// Special handling for "env = VARNAME, value" pattern (common in Hyprland and other configs)
	if strings.ToLower(keyword) == "env" {
		envPattern := regexp.MustCompile(`(?i)env\s*=\s*([A-Z_][A-Z0-9_]*)\s*,\s*(.+)`)
		if match := envPattern.FindStringSubmatch(line); len(match) > 2 {
			// match[1] is the env var name, match[2] is the value
			// We want to check the value, not the variable name
			values = append(values, strings.TrimSpace(match[2]))
			return values
		}
	}

	// Look for patterns like keyword=value or keyword: value
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)` + keyword + `['"]*\s*[=:]\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)` + keyword + `['"]*\s*[=:]\s*'([^']+)'`),
		regexp.MustCompile(`(?i)` + keyword + `['"]*\s*[=:]\s*([^\s"',;]+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				values = append(values, match[1])
			}
		}
	}

	return values
}

func (d *Detector) calculatePatternConfidence(secretType SecretType, value, context string) float64 {
	// Adjust based on secret type
	var base float64
	switch secretType {
	case SecretTypePrivateKey:
		base = 0.95
	case SecretTypeAWSKey:
		base = 0.9
	case SecretTypeJWT:
		base = 0.85
	case SecretTypeDatabaseURL:
		base = 0.85
	case SecretTypeCreditCard:
		base = 0.9
	case SecretTypePassword:
		base = 0.7
	default:
		base = 0.75
	}

	// Adjust based on context
	if d.hasSecretContext(context) {
		base += 0.1
	}

	// Adjust based on value characteristics
	if len(value) > 40 {
		base += 0.05
	}

	if d.isPlaceholder(value) {
		base -= 0.5
	}

	return min(base, 1.0)
}

func (d *Detector) calculateContextConfidence(keyword, value, context string) float64 {
	base := 0.6

	// Higher confidence for certain keywords
	highConfidenceKeywords := []string{"secret", "key", "token", "password", "private"}
	for _, hck := range highConfidenceKeywords {
		if strings.Contains(strings.ToLower(keyword), hck) {
			base = 0.8
			break
		}
	}

	// Analyze context for additional clues
	contextLower := strings.ToLower(context)

	// Look for security-related keywords in surrounding context
	securityKeywords := []string{"auth", "credential", "secret", "private", "config", "env"}
	for _, secWord := range securityKeywords {
		if strings.Contains(contextLower, secWord) {
			base += 0.1
			break
		}
	}

	// Look for warning comments that might indicate it's a placeholder
	placeholderIndicators := []string{"example", "placeholder", "dummy", "sample", "test", "todo", "fixme"}
	for _, placeholder := range placeholderIndicators {
		if strings.Contains(contextLower, placeholder) {
			base -= 0.2
			break
		}
	}

	// Adjust based on value entropy
	entropy := CalculateEntropy(value)
	if entropy > MediumEntropyThreshold {
		base += 0.2
	}

	// Adjust based on value length
	if len(value) > 20 {
		base += 0.1
	}

	return min(base, 1.0)
}

func (d *Detector) inferSecretType(keyword, value string) SecretType {
	lower := strings.ToLower(keyword)
	valueLower := strings.ToLower(value)

	// First check the value itself for specific patterns
	switch {
	case strings.HasPrefix(valueLower, "akia"):
		return SecretTypeAWSKey
	case strings.HasPrefix(valueLower, "eyj"):
		return SecretTypeJWT
	case strings.HasPrefix(valueLower, "sk-") || strings.HasPrefix(valueLower, "pk-"):
		return SecretTypeAPIKey
	case strings.HasPrefix(valueLower, "ghp_") || strings.HasPrefix(valueLower, "gho_"):
		return SecretTypeGitHubToken
	case strings.Contains(valueLower, "://") && (strings.Contains(valueLower, "postgres") || strings.Contains(valueLower, "mysql") || strings.Contains(valueLower, "mongodb")):
		return SecretTypeDatabaseURL
	case strings.HasPrefix(value, "-----BEGIN") && strings.Contains(value, "PRIVATE KEY"):
		return SecretTypePrivateKey
	}

	// Then check the keyword for context
	switch {
	case strings.Contains(lower, "password") || strings.Contains(lower, "pwd"):
		return SecretTypePassword
	case strings.Contains(lower, "aws") || strings.Contains(lower, "access"):
		return SecretTypeAWSKey
	case strings.Contains(lower, "github") || strings.Contains(lower, "gh"):
		return SecretTypeGitHubToken
	case strings.Contains(lower, "jwt") || strings.Contains(lower, "bearer"):
		return SecretTypeJWT
	case strings.Contains(lower, "database") || strings.Contains(lower, "db"):
		return SecretTypeDatabaseURL
	case strings.Contains(lower, "private") || strings.Contains(lower, "rsa") || strings.Contains(lower, "ssh"):
		return SecretTypePrivateKey
	case strings.Contains(lower, "key"):
		return SecretTypeAPIKey
	case strings.Contains(lower, "token"):
		return SecretTypeToken
	case strings.Contains(lower, "secret"):
		return SecretTypeGeneric
	default:
		return SecretTypeGeneric
	}
}

func (d *Detector) getRiskLevel(secretType SecretType, confidence float64) RiskLevel {
	// Base risk level by secret type
	baseRisk := map[SecretType]RiskLevel{
		SecretTypePrivateKey:  RiskLevelCritical,
		SecretTypeAWSKey:      RiskLevelCritical,
		SecretTypeDatabaseURL: RiskLevelHigh,
		SecretTypePassword:    RiskLevelHigh,
		SecretTypeAPIKey:      RiskLevelHigh,
		SecretTypeToken:       RiskLevelMedium,
		SecretTypeJWT:         RiskLevelMedium,
		SecretTypeGeneric:     RiskLevelMedium,
		SecretTypeEmail:       RiskLevelLow,
		SecretTypeIPAddress:   RiskLevelLow,
	}

	risk := baseRisk[secretType]

	// Adjust based on confidence
	if confidence < 0.6 {
		if risk > RiskLevelLow {
			risk--
		}
	} else if confidence > 0.9 {
		if risk < RiskLevelCritical {
			risk++
		}
	}

	return risk
}

func (d *Detector) hasSecretContext(context string) bool {
	lower := strings.ToLower(context)
	secretIndicators := []string{"secret", "private", "confidential", "internal", "auth", "cred"}

	for _, indicator := range secretIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

func (d *Detector) extractContext(text, value string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, value) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func (d *Detector) deduplicateFindings(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	// Group findings by location (file + line + similar content)
	groups := make(map[string][]Finding)

	for _, finding := range findings {
		// Create a key based on file, line, and a normalized version of the content
		key := finding.Location.FilePath + ":" + fmt.Sprintf("%d", finding.Location.LineNumber)
		groups[key] = append(groups[key], finding)
	}

	var unique []Finding

	// For each location group, find overlapping findings and keep the best one
	for _, group := range groups {
		if len(group) == 1 {
			unique = append(unique, group[0])
			continue
		}

		// Multiple findings at same location - deduplicate overlapping ones
		deduplicated := d.deduplicateOverlapping(group)
		unique = append(unique, deduplicated...)
	}

	return unique
}

func (d *Detector) deduplicateOverlapping(findings []Finding) []Finding {
	var result []Finding

	for i, finding := range findings {
		isOverlapped := false

		for j, other := range findings {
			if i == j {
				continue
			}

			// Check if one finding's value is contained in another
			if d.isOverlapping(finding, other) {
				// Keep the more specific one or the one with higher confidence
				if d.shouldPrefer(other, finding) {
					isOverlapped = true
					break
				}
			}
		}

		if !isOverlapped {
			result = append(result, finding)
		}
	}

	return result
}

func (d *Detector) isOverlapping(f1, f2 Finding) bool {
	// Check if one value is contained in the other
	v1Contains2 := strings.Contains(f1.RawValue, f2.RawValue)
	v2Contains1 := strings.Contains(f2.RawValue, f1.RawValue)

	// If they have significant overlap (one contains the other), they're overlapping
	if v1Contains2 || v2Contains1 {
		return true
	}

	// Check if they're from the same line and have similar start positions
	if f1.Location.LineNumber == f2.Location.LineNumber {
		// Calculate the position of each value in the line
		pos1 := strings.Index(f1.Location.LineText, f1.RawValue)
		pos2 := strings.Index(f2.Location.LineText, f2.RawValue)

		// If they're within a small range, they might be overlapping parts of the same secret
		if pos1 >= 0 && pos2 >= 0 && abs(pos1-pos2) < 20 {
			return true
		}
	}

	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (d *Detector) shouldPrefer(f1, f2 Finding) bool {
	// Prefer more specific types over generic ones
	if d.isMoreSpecific(f1.Type, f2.Type) {
		return true
	}
	if d.isMoreSpecific(f2.Type, f1.Type) {
		return false
	}

	// If types are equally specific, prefer higher confidence
	if f1.Confidence > f2.Confidence {
		return true
	}

	// If confidence is equal, prefer longer matches (more complete)
	if f1.Confidence == f2.Confidence {
		return len(f1.RawValue) > len(f2.RawValue)
	}

	return false
}

// isMoreSpecific returns true if type1 is more specific than type2
func (d *Detector) isMoreSpecific(type1, type2 SecretType) bool {
	// Define specificity order (higher number = more specific)
	specificity := map[SecretType]int{
		SecretTypeGeneric:       0,
		SecretTypeGenericAPIKey: 1,
		SecretTypeToken:         2,
		SecretTypePassword:      3,
		SecretTypeEmail:         3,
		SecretTypeIPAddress:     3,
		SecretTypeCreditCard:    4,
		SecretTypeSSN:           4,
		SecretTypePII:           4,
		SecretTypeJWT:           5,
		SecretTypeDatabaseURL:   5,
		SecretTypeGitHubToken:   6,
		SecretTypeAWSKey:        7,
		SecretTypeAnthropicKey:  8,
		SecretTypeAPIKey:        8,
		SecretTypePrivateKey:    9,
	}

	spec1, ok1 := specificity[type1]
	spec2, ok2 := specificity[type2]

	if !ok1 {
		spec1 = 0
	}
	if !ok2 {
		spec2 = 0
	}

	return spec1 > spec2
}

func (d *Detector) SetSensitivity(level SensitivityLevel) {
	switch level {
	case SensitivityLow:
		d.entropyAnalyzer.threshold = LowEntropyThreshold
	case SensitivityMedium:
		d.entropyAnalyzer.threshold = MediumEntropyThreshold
	case SensitivityHigh:
		d.entropyAnalyzer.threshold = HighEntropyThreshold
	case SensitivityParanoid:
		d.entropyAnalyzer.threshold = 3.0
	}
}

// Utility functions

func extractQuotedStrings(line string) []string {
	var strings []string

	// Extract double-quoted strings (including empty ones)
	doubleQuoted := regexp.MustCompile(`"([^"]*)"`)
	matches := doubleQuoted.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			strings = append(strings, match[1])
		}
	}

	// Extract single-quoted strings (including empty ones)
	singleQuoted := regexp.MustCompile(`'([^']*)'`)
	matches = singleQuoted.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			strings = append(strings, match[1])
		}
	}

	return strings
}

func redactValue(value string) string {
	if len(value) <= 8 {
		return "********"
	}

	visible := 4
	if len(value) > 20 {
		visible = 6
	}

	return value[:visible] + "****" + value[len(value)-visible:]
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (d *Detector) isCommonConfigPattern(candidate, context string) bool {
	lower := strings.ToLower(candidate)
	lowerContext := strings.ToLower(context)

	// Check if this is an environment variable NAME (not value) in an assignment
	// Patterns like: env = VARNAME, value or export VARNAME=value
	if strings.HasPrefix(lowerContext, "env =") || strings.HasPrefix(lowerContext, "export ") {
		// Extract the part after "env =" to check structure
		parts := strings.Split(context, ",")
		if len(parts) >= 2 {
			// This is likely "env = VARNAME, value" pattern
			varPart := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(lowerContext, "env ="), "export "))
			// If candidate matches the variable name part (before comma)
			if strings.HasPrefix(varPart, strings.ToLower(candidate)) {
				return true
			}
		}

		// If candidate looks like an env var name (uppercase with underscores)
		if regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(candidate) {
			return true
		}
	}

	// Common environment variable name patterns (generic, not hardcoded)
	// These are patterns that indicate configuration, not secrets
	envPatterns := []string{
		"_SIZE", "_TYPE", "_PLATFORM", "_THEME", "_PATH", "_HOME", "_DIR",
		"_DISPLAY", "_SESSION", "_DESKTOP", "_DRIVER", "_DEVICE", "_BACKEND",
		"_VERSION", "_MODULE", "_HINT", "_MODE", "_LEVEL", "_DEBUG",
		"_CONFIG", "_SETTING", "_OPTION", "_FEATURE", "_FLAG", "_ENABLED",
		"_DISABLED", "_COLOR", "_STYLE", "_FORMAT", "_LAYOUT", "_SCHEMA",
		"_RUNTIME", "_BUILD", "_ARCH", "_OS", "_SYSTEM", "_SERVICE",
		"_SERVER", "_CLIENT", "_HOST", "_PORT", "_PROTOCOL", "_INTERFACE",
		"_LOCALE", "_LANG", "_REGION", "_ZONE", "_ENCODING", "_CHARSET",
	}

	upperCandidate := strings.ToUpper(candidate)
	for _, pattern := range envPatterns {
		if strings.HasSuffix(upperCandidate, pattern) {
			return true
		}
	}

	// Common prefixes for non-secret env vars
	envPrefixes := []string{
		"XDG_", "GTK_", "QT_", "KDE_", "GNOME_", "MESA_", "GL_", "VK_",
		"WAYLAND_", "X11_", "DRI_", "EGL_", "WLR_", "AQ_", "ELECTRON_",
		"NODE_", "NPM_", "PYTHON_", "JAVA_", "GO_", "RUST_", "CARGO_",
		"CMAKE_", "PKG_", "LD_", "LC_", "LANG_", "TZ_", "TERM_",
		"SHELL_", "EDITOR_", "PAGER_", "BROWSER_", "MAIL_", "GPG_",
		"SSH_", "DBUS_", "SYSTEMD_", "PULSE_", "ALSA_", "JACK_",
		"SDL_", "STEAM_", "WINE_", "PROTON_", "VULKAN_", "CUDA_",
		"OPENCL_", "HIP_", "ROC_", "SYCL_", "ONEAPI_", "MKL_",
	}

	for _, prefix := range envPrefixes {
		if strings.HasPrefix(upperCandidate, prefix) {
			return true
		}
	}

	// System paths
	pathPatterns := []string{
		"/usr/", "/var/", "/etc/", "/home/", "/opt/", "/tmp/", "/bin/", "/sbin/",
		"/.config/", "/.local/", "/.cache/", "~/.config/", "~/.local/",
		"c:\\", "program files", "appdata", "documents",
		"/dev/dri/", "/sys/", "/proc/", "/run/", "/mnt/", "/media/",
	}

	for _, pattern := range pathPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Package/namespace identifiers
	if strings.HasPrefix(lower, "com.") || strings.HasPrefix(lower, "org.") ||
		strings.HasPrefix(lower, "net.") || strings.HasPrefix(lower, "io.") ||
		strings.HasPrefix(lower, "co.") || strings.HasPrefix(lower, "de.") {
		return true
	}

	// Common Wayland/X11/GPU values
	waylandPatterns := []string{
		"wayland", "x11", "xcb", "xwayland", "vulkan", "opengl", "egl",
		"nvidia", "amd", "intel", "radeon", "nouveau", "mesa", "drm",
		"gbm", "kms", "card0", "card1", "renderD128", "renderD129",
		"hyprland", "sway", "wlroots", "qt5ct", "kvantum", "gtk",
	}

	for _, pattern := range waylandPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Color patterns
	colorPatterns := []string{
		"color", "rgb", "rgba", "hex", "#", "red", "green", "blue", "yellow",
		"cyan", "magenta", "black", "white", "gray", "grey", "purple",
		"orange", "pink", "brown", "bold", "italic", "underline", "normal",
		"bright", "dim", "foreground", "background", "fg", "bg",
	}

	for _, pattern := range colorPatterns {
		if strings.Contains(lower, pattern) || strings.Contains(lowerContext, pattern) {
			return true
		}
	}

	// Shell/terminal patterns
	shellPatterns := []string{
		"bash", "zsh", "fish", "sh", "cmd", "powershell", "terminal",
		"console", "completion", "function", "alias", "export",
		"echo", "printf", "source", "eval", "exec", "which", "where",
	}

	for _, pattern := range shellPatterns {
		if strings.Contains(lowerContext, pattern) {
			return true
		}
	}

	// File extensions and formats
	if strings.Contains(lower, ".fish") || strings.Contains(lower, ".sh") ||
		strings.Contains(lower, ".py") || strings.Contains(lower, ".js") ||
		strings.Contains(lower, ".json") || strings.Contains(lower, ".xml") ||
		strings.Contains(lower, ".yaml") || strings.Contains(lower, ".toml") {
		return true
	}

	// Common configuration keys/values
	configPatterns := []string{
		"settings", "config", "options", "preferences", "profile",
		"theme", "layout", "display", "format", "style", "appearance",
		"font", "size", "family", "weight", "editor", "viewer",
		"application", "program", "software", "system", "service",
		"version", "release", "build", "revision", "timeout", "interval",
		"maximum", "minimum", "default", "enabled", "disabled",
		"favorites", "bookmarks", "history", "recent", "cache",
		"true", "false", "yes", "no", "on", "off", "auto", "manual",
		"none", "all", "some", "any", "both", "either", "neither",
	}

	for _, pattern := range configPatterns {
		if strings.Contains(lower, pattern) || strings.Contains(lowerContext, pattern) {
			return true
		}
	}

	// Unix socket paths
	if strings.Contains(lower, "unix:") || strings.Contains(lower, ".sock") {
		return true
	}

	// Environment variable patterns (but not secret ones)
	if strings.HasPrefix(candidate, "$") {
		return true
	}

	// Numeric values and common resolutions (but not credit card numbers)
	if regexp.MustCompile(`^\d+x\d+$|^\d+:\d+$|^\d+\.\d+$`).MatchString(candidate) {
		return true
	}

	// Don't consider long sequences of digits as config if they might be sensitive
	// Credit cards are typically 13-19 digits
	if len(candidate) >= 13 && len(candidate) <= 19 && regexp.MustCompile(`^\d+$`).MatchString(candidate) {
		return false
	}

	// GPU device IDs and PCI addresses
	if regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{4}$|^[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]$`).MatchString(candidate) {
		return true
	}

	return false
}

func (d *Detector) hasSecretCharacteristics(candidate string) bool {
	// Check for characteristics that are typical of secrets
	// but not of regular configuration values

	// Very high entropy with mixed case and numbers suggests API keys
	entropy := CalculateEntropy(candidate)
	if entropy > 5.0 && len(candidate) > 30 {
		var hasUpper, hasLower, hasDigit bool
		for _, char := range candidate {
			if char >= 'A' && char <= 'Z' {
				hasUpper = true
			} else if char >= 'a' && char <= 'z' {
				hasLower = true
			} else if char >= '0' && char <= '9' {
				hasDigit = true
			}
		}
		if hasUpper && hasLower && hasDigit {
			return true
		}
	}

	// Known secret patterns - always flag these
	lower := strings.ToLower(candidate)
	if strings.HasPrefix(lower, "sk-") || strings.HasPrefix(lower, "pk-") ||
		strings.HasPrefix(lower, "akia") || strings.HasPrefix(lower, "eyj") ||
		strings.HasPrefix(lower, "ghp_") || strings.HasPrefix(lower, "gho_") ||
		strings.HasPrefix(lower, "sk-ant-api03-") {
		return true
	}

	// Base64-like patterns (but not if they contain obvious words)
	if len(candidate) > 20 && len(candidate)%4 == 0 {
		// Check if it's mostly alphanumeric with occasional = or +
		base64Chars := 0
		for _, char := range candidate {
			if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') || char == '+' || char == '/' || char == '=' {
				base64Chars++
			}
		}
		if float64(base64Chars)/float64(len(candidate)) > 0.9 {
			// Looks like base64, but check if it contains obvious words
			if !d.containsDictionaryWords(candidate) {
				return true
			}
		}
	}

	return false
}
