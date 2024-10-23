package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CopyDirectory(src, dst string, ignores []string) error {
	// First, remove any ignored files in the destination
	if err := RemoveIgnoredFiles(dst, ignores); err != nil {
		return fmt.Errorf("error removing ignored files: %v", err)
	}

	// Check if the directory itself should be ignored
	if shouldIgnore(src, ignores) {
		return nil
	}

	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading source directory: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() == ".git" || entry.Name() == ".github" {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if shouldIgnore(srcPath, ignores) {
			continue
		}

		if entry.IsDir() {
			if err := CopyDirectory(srcPath, dstPath, ignores); err != nil {
				return fmt.Errorf("error copying directory: %v", err)
			}
		} else {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			if err := CopyFile(srcPath, dstPath, ignores); err != nil {
				return fmt.Errorf("error copying file: %v", err)
			}
		}
	}

	return nil
}

func CopyFile(src, dst string, ignores []string) error {
	// First, remove any ignored files in the destination
	if err := RemoveIgnoredFiles(dst, ignores); err != nil {
		return fmt.Errorf("error removing ignored files: %v", err)
	}

	// Check if the file should be ignored
	if shouldIgnore(src, ignores) {
		return nil // Skip this file without error
	}

	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer dstFile.Close()

	// Copy the file contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file contents: %v", err)
	}

	// Preserve the file mode
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting source file info: %v", err)
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// CopyPath copies a file or directory from src to dst
func CopyPath(src, dst string, ignores []string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting source info: %w", err)
	}

	if srcInfo.IsDir() {
		return CopyDirectory(src, dst, ignores)
	}
	return CopyFile(src, dst, ignores)
}

func shouldIgnore(path string, ignores []string) bool {
	// Convert path to use forward slashes for consistency
	path = filepath.ToSlash(path)

	for _, ignore := range ignores {
		// Convert ignore pattern to use forward slashes
		ignore = filepath.ToSlash(ignore)

		// Handle exact matches first (like .gitignore)
		if filepath.Base(path) == ignore {
			return true
		}

		// Handle directory-specific patterns (ending with /)
		if strings.HasSuffix(ignore, "/") {
			pattern := strings.TrimSuffix(ignore, "/")

			// Handle **/ pattern (matches any level of directories)
			if strings.HasPrefix(pattern, "**") {
				pattern = strings.TrimPrefix(pattern, "**/")
				if strings.Contains(path, pattern) {
					return true
				}
				continue
			}

			// Handle directory patterns by checking each path component
			dirs := strings.Split(path, "/")
			for _, dir := range dirs {
				if matched, _ := filepath.Match(pattern, dir); matched {
					return true
				}
			}
			continue
		}

		// Handle full path matching (e.g., node_modules/**)
		if strings.HasSuffix(ignore, "/**") {
			prefix := strings.TrimSuffix(ignore, "/**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
			continue
		}

		// Handle file patterns with multiple extensions (e.g., *.{jpg,png})
		if strings.Contains(ignore, "{") && strings.Contains(ignore, "}") {
			startBrace := strings.Index(ignore, "{")
			endBrace := strings.Index(ignore, "}")
			if startBrace > 0 && endBrace > startBrace {
				prefix := ignore[:startBrace]
				extsStr := ignore[startBrace+1 : endBrace]
				exts := strings.Split(extsStr, ",")
				base := filepath.Base(path)

				for _, ext := range exts {
					pattern := prefix + strings.TrimSpace(ext)
					if matched, _ := filepath.Match(pattern, base); matched {
						return true
					}
				}
			}
			continue
		}

		// Try to match against the full path first
		if matched, _ := filepath.Match(ignore, path); matched {
			return true
		}

		// Then try to match against the base name
		base := filepath.Base(path)
		if matched, _ := filepath.Match(ignore, base); matched {
			return true
		}
	}

	return false
}

// RemoveIgnoredFiles removes files and directories that match the ignore patterns
func RemoveIgnoredFiles(dst string, ignores []string) error {
	return filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		// if the error is no such file or directory, we can ignore it
		if err != nil && os.IsNotExist(err) {
			return nil
		}

		// return other errors
		if err != nil {
			return err
		}

		if shouldIgnore(path, ignores) {
			if info.IsDir() {
				// Remove directory and all its contents
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("error removing ignored directory: %v", err)
				}
				return filepath.SkipDir // Skip the removed directory
			}
			// Remove single file
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("error removing ignored file: %v", err)
			}
		}
		return nil
	})
}
