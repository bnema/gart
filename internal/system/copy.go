package system

import (
	"io"
	"os"
	"path/filepath"
)

func CopyDirectory(src, dst string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return err
	}

	// Read the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
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
				return err
			}
		} else {
			// Skip symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			// Copy regular files
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyFile(src, dst string) error {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the file contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Preserve the file mode
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}
