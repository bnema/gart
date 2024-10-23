package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		ignores  []string
		expected bool
	}{
		{
			name:     "Simple file ignore",
			path:     "/test/file.json",
			ignores:  []string{"*.json"},
			expected: true,
		},
		{
			name:     "Directory ignore",
			path:     "/test/cache/file.txt",
			ignores:  []string{"cache/"},
			expected: true,
		},
		{
			name:     "Wildcard directory",
			path:     "/test/temp_cache/file.txt",
			ignores:  []string{"temp_*/"},
			expected: true,
		},
		{
			name:     "Nested directory",
			path:     "/test/src/cache/file.txt",
			ignores:  []string{"**/cache/"},
			expected: true,
		},
		{
			name:     "Multiple extensions",
			path:     "/test/image.jpg",
			ignores:  []string{"*.{jpg,png}"},
			expected: true,
		},
		{
			name:     "Non-matching path",
			path:     "/test/document.txt",
			ignores:  []string{"*.json", "cache/", "*.{jpg,png}"},
			expected: false,
		},
		{
			name:     "Directory itself should be ignored",
			path:     "/test",
			ignores:  []string{"test/"},
			expected: true,
		},
		{
			name:     "Directory with trailing slash",
			path:     "/home/user/dotfiles/test",
			ignores:  []string{"test/"},
			expected: true,
		},
		{
			name:     "Directory content with trailing slash",
			path:     "/home/user/dotfiles/test/something.txt",
			ignores:  []string{"test/"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnore(tt.path, tt.ignores)
			assert.Equal(t, tt.expected, result)
		})
	}
}
