package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CopyDirectory(src, dst string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading source directory: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() == ".git" || entry.Name() == ".github" {
			// Skip .git and .github directories
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := CopyDirectory(srcPath, dstPath); err != nil {
				return fmt.Errorf("error copying directory: %v", err)
			}
		} else {
			// Skip symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			// Copy regular files
			if err := CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("error copying file: %v", err)
			}
		}
	}

	return nil
}

func CopyFile(src, dst string) error {
	// Ensure the destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}

	// Check if destination already exists
	if _, err := os.Stat(dst); err == nil {
		// Destination exists, remove it if it's a directory
		if info, err := os.Stat(dst); err == nil && info.IsDir() {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("error removing existing directory at destination: %v", err)
			}
		}
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
