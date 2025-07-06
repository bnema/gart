package security

import (
	"testing"
)

func TestFalsePositives(t *testing.T) {
	// Initialize the detector with default config
	config := DefaultSecurityConfig()
	detector := NewDetector(config)

	// Test cases that were reported as false positives
	testCases := []struct {
		name         string
		content      string
		shouldDetect bool
	}{
		{
			name:         "Cosmic config",
			content:      `com.system76.CosmicSettings`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 1",
			content:      `Don't anything`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 2",
			content:      `Server paths`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 3",
			content:      `Substitution ment`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 4",
			content:      ` | str\\s*`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 5",
			content:      `Write rn v1)`,
			shouldDetect: false,
		},
		{
			name:         "Fish completion 6",
			content:      `Don't encies`,
			shouldDetect: false,
		},
		{
			name:         "Fish config 1",
			content:      `Downlo dex...`,
			shouldDetect: false,
		},
		{
			name:         "Fish config 2",
			content:      `$HOME/ v.fish`,
			shouldDetect: false,
		},
		{
			name:         "Fish config 3",
			content:      `env NV " nvim`,
			shouldDetect: false,
		},
		{
			name:         "Fish config 4",
			content:      `unix:/ n.sock`,
			shouldDetect: false,
		},
		{
			name:         "Fish config 5",
			content:      `///run n.sock`,
			shouldDetect: false,
		},
		{
			name:         "Fish variables 1",
			content:      `cyan bold`,
			shouldDetect: false,
		},
		{
			name:         "Fish variables 2",
			content:      `white rblack`,
			shouldDetect: false,
		},
		{
			name:         "Fish variables 3",
			content:      `normal erline`,
			shouldDetect: false,
		},
		{
			name:         "Fish variables 4",
			content:      `brwhit 3dcyan`,
			shouldDetect: false,
		},
		{
			name:         "Fish variables 5",
			content:      `/home/ al/bin`,
			shouldDetect: false,
		},
		{
			name:         "OpenAI key (should be detected)",
			content:      `OPENAI_API_KEY=sk-1234567890abcdefghijklmnopqrstuvwxyz`,
			shouldDetect: true,
		},
		{
			name:         "Anthropic key (should be detected)",
			content:      `ANTHROPIC_API_KEY=sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmno`,
			shouldDetect: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			findings := detector.DetectSecrets([]byte(tc.content), "test.txt")

			if tc.shouldDetect {
				if len(findings) == 0 {
					t.Errorf("Expected to detect secrets but found none")
				}
			} else {
				if len(findings) > 0 {
					t.Errorf("Expected no secrets but found %d: %v", len(findings), findings)
				}
			}
		})
	}
}
