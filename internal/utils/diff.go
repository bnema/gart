package utils

import (
	"os"
	"path/filepath"

	"github.com/bnema/Gart/internal/system"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffFiles compares two files or directories and returns true if they are different
func DiffFiles(origin, dest string) (bool, error) {
	dmp := diffmatchpatch.New()

	var diff func(string, string) (bool, error)
	diff = func(origin, dest string) (bool, error) {
		info1, err := os.Stat(origin)
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

		info2, err := os.Stat(dest)
		if os.IsNotExist(err) {
			// The file exists in path1 but not in path2
			// Copy the file from path1 to path2
			if err := system.CopyFile(origin, dest); err != nil {
				return false, err
			}
			return true, nil
		} else if err != nil {
			return false, err
		}

		if info1.IsDir() && info2.IsDir() {
			filesOrigin, err := os.ReadDir(origin)
			if err != nil {
				return false, err
			}

			changed := false
			for _, fileOrig := range filesOrigin {
				if fileOrig.Name() == ".git" || fileOrig.Name() == ".github" {
					// Skip .git and .github directories
					continue
				}

				filePathOrig := filepath.Join(origin, fileOrig.Name())
				filePathDest := filepath.Join(dest, fileOrig.Name())

				fileChanged, err := diff(filePathOrig, filePathDest)
				if err != nil {
					return false, err
				}
				if fileChanged {
					changed = true
				}
			}
			return changed, nil
		} else if !info1.IsDir() && !info2.IsDir() {
			contentOrig, err := os.ReadFile(origin)
			if err != nil {
				return false, err
			}

			contentDest, err := os.ReadFile(dest)
			if err != nil {
				return false, err
			}

			diffs := dmp.DiffMain(string(contentOrig), string(contentDest), false)
			if len(diffs) > 1 {
				// Differences found, copy the file from path1 to path2
				if err := system.CopyFile(origin, dest); err != nil {
					return false, err
				}
				return true, nil
			}
		}

		return false, nil
	}

	changed, err := diff(origin, dest)
	if err != nil {
		return false, err
	}
	return changed, nil
}
