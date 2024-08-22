package system

import (
	"os"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffFiles compares files or directories at origin and dest paths.
// It returns true if changes were made, and false otherwise.
func DiffFiles(origin, dest string) (bool, error) {
	dmp := diffmatchpatch.New()
	return diffRecursive(origin, dest, dmp)
}

// diffRecursive is a helper function that recursively compares files and directories.
func diffRecursive(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch) (bool, error) {
	originInfo, err := os.Stat(origin)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	// Skip .git and .github directories
	if originInfo.IsDir() && (originInfo.Name() == ".git" || originInfo.Name() == ".github") {
		return false, nil
	}

	destInfo, err := os.Stat(dest)
	if os.IsNotExist(err) {
		return copyItem(origin, dest, originInfo.IsDir())
	} else if err != nil {
		return false, err
	}

	if originInfo.IsDir() && destInfo.IsDir() {
		return diffDirectories(origin, dest, dmp)
	} else if !originInfo.IsDir() && !destInfo.IsDir() {
		return diffFiles(origin, dest, dmp)
	} else {
		// One is a file, the other is a directory
		return replaceItem(origin, dest, originInfo.IsDir())
	}
}

// diffDirectories compares the contents of two directories.
func diffDirectories(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch) (bool, error) {
	originFiles, err := os.ReadDir(origin)
	if err != nil {
		return false, err
	}

	destFiles, err := os.ReadDir(dest)
	if err != nil {
		return false, err
	}

	changed := false

	// Create maps to track files in both directories
	originMap := make(map[string]os.DirEntry)
	destMap := make(map[string]os.DirEntry)

	for _, file := range originFiles {
		if file.Name() == ".git" || file.Name() == ".github" {
			continue
		}
		originMap[file.Name()] = file
	}

	for _, file := range destFiles {
		if file.Name() == ".git" || file.Name() == ".github" {
			continue
		}
		destMap[file.Name()] = file
	}

	// Check for new or modified files in origin
	for name, file := range originMap {
		originPath := filepath.Join(origin, name)
		destPath := filepath.Join(dest, name)

		if _, exists := destMap[name]; !exists {
			// File is new in origin
			fileChanged, err := copyItem(originPath, destPath, file.IsDir())
			if err != nil {
				return false, err
			}
			changed = changed || fileChanged
		} else {
			// File exists in both, check for changes
			fileChanged, err := diffRecursive(originPath, destPath, dmp)
			if err != nil {
				return false, err
			}
			changed = changed || fileChanged
		}
	}

	// Check for deleted files
	for name := range destMap {
		if _, exists := originMap[name]; !exists {
			// File exists in dest but not in origin, so it was deleted
			err := os.RemoveAll(filepath.Join(dest, name))
			if err != nil {
				return false, err
			}
			changed = true
		}
	}

	return changed, nil
}

// diffFiles compares the contents of two files.
func diffFiles(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch) (bool, error) {
	originContent, err := os.ReadFile(origin)
	if err != nil {
		return false, err
	}
	destContent, err := os.ReadFile(dest)
	if err != nil {
		return false, err
	}

	diffs := dmp.DiffMain(string(originContent), string(destContent), false)
	if len(diffs) > 1 {
		return copyItem(origin, dest, false)
	}
	return false, nil
}

// copyItem copies a file or directory from origin to dest.
func copyItem(origin, dest string, isDir bool) (bool, error) {
	if isDir {
		err := CopyDirectory(origin, dest)
		return true, err
	}
	err := CopyFile(origin, dest)
	return true, err
}

// replaceItem removes the destination item and replaces it with the origin item.
func replaceItem(origin, dest string, isDir bool) (bool, error) {
	err := os.RemoveAll(dest)
	if err != nil {
		return false, err
	}
	return copyItem(origin, dest, isDir)
}
