package utils

import (
	"os"
	"path/filepath"

	"github.com/bnema/Gart/system"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffFiles compares two files or directories and returns true if they are different
func DiffFiles(path1, path2 string) (bool, error) {
	// p1 = origin
	// p2 = destination
	dmp := diffmatchpatch.New()

	var diff func(string, string) (bool, error)
	diff = func(p1, p2 string) (bool, error) {
		info1, err := os.Stat(p1)
		if os.IsNotExist(err) {
			// Skip the comparison if the file or directory doesn't exist in path1
			return false, nil
		} else if err != nil {
			return false, err
		}

		if info1.IsDir() {
			if info1.Name() == ".git" || info1.Name() == ".github" {
				// Skip .git and .github directories
				return false, nil
			}
		}

		info2, err := os.Stat(p2)
		if os.IsNotExist(err) {
			// The file exists in path1 but not in path2
			// Copy the file from path1 to path2
			if err := system.CopyFile(p1, p2); err != nil {
				return false, err
			}
			return true, nil
		} else if err != nil {
			return false, err
		}

		if info1.IsDir() && info2.IsDir() {
			files1, err := os.ReadDir(p1)
			if err != nil {
				return false, err
			}

			changed := false
			for _, file1 := range files1 {
				if file1.Name() == ".git" || file1.Name() == ".github" {
					// Skip .git and .github directories
					continue
				}

				filePath1 := filepath.Join(p1, file1.Name())
				filePath2 := filepath.Join(p2, file1.Name())

				fileChanged, err := diff(filePath1, filePath2)
				if err != nil {
					return false, err
				}
				if fileChanged {
					changed = true
				}
			}
			return changed, nil
		} else if !info1.IsDir() && !info2.IsDir() {
			content1, err := os.ReadFile(p1)
			if err != nil {
				return false, err
			}

			content2, err := os.ReadFile(p2)
			if err != nil {
				return false, err
			}

			diffs := dmp.DiffMain(string(content1), string(content2), false)
			if len(diffs) > 1 {
				// Differences found, copy the file from path1 to path2
				if err := system.CopyFile(p1, p2); err != nil {
					return false, err
				}
				return true, nil
			}
		}

		return false, nil
	}

	changed, err := diff(path1, path2)
	if err != nil {
		return false, err
	}
	return changed, nil
}
