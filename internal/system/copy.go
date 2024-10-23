package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CopyDirectory(src, dst string, ignores []string) error {
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
	for _, ignore := range ignores {
		if matched, _ := filepath.Match(ignore, filepath.Base(path)); matched {
			return true
		}
		if strings.HasPrefix(path, ignore) {
			return true
		}
	}
	return false
}
