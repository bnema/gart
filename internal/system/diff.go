package system

import (
	"os"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffFiles compares files or directories based on the sync mode.
// If reverseSyncMode is true, the destination is considered the source.
func DiffFiles(origin, dest string, ignores []string, reverseSyncMode bool) (bool, error) {
	dmp := diffmatchpatch.New()

	if reverseSyncMode {
		// Pull mode (reverse sync)
		return diffRecursive(dest, origin, dmp, ignores)
	}
	// Push mode (default)
	return diffRecursive(origin, dest, dmp, ignores)
}

func diffRecursive(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch, ignores []string) (bool, error) {
	// Check if the path should be ignored before doing anything else
	if shouldIgnore(origin, ignores) {
		return false, nil
	}

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
		// Only copy if the item isn't ignored
		if !shouldIgnore(origin, ignores) {
			return copyItem(origin, dest, originInfo.IsDir(), ignores)
		}
		return false, nil
	} else if err != nil {
		return false, err
	}

	if originInfo.IsDir() && destInfo.IsDir() {
		return diffDirectories(origin, dest, dmp, ignores)
	} else if !originInfo.IsDir() && !destInfo.IsDir() {
		if !shouldIgnore(origin, ignores) {
			return diffFiles(origin, dest, dmp, ignores)
		}
		return false, nil
	} else {
		// One is a file, the other is a directory
		if !shouldIgnore(origin, ignores) {
			return replaceItem(origin, dest, originInfo.IsDir(), ignores)
		}
		return false, nil
	}
}

// diffDirectories compares the contents of two directories.
func diffDirectories(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch, ignores []string) (bool, error) {
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
		originPath := filepath.Join(origin, file.Name())
		if !shouldIgnore(originPath, ignores) {
			originMap[file.Name()] = file
		}
	}

	for _, file := range destFiles {
		if file.Name() == ".git" || file.Name() == ".github" {
			continue
		}
		destPath := filepath.Join(dest, file.Name())
		if !shouldIgnore(destPath, ignores) {
			destMap[file.Name()] = file
		}
	}

	// Check for new or modified files in origin
	for name, file := range originMap {
		originPath := filepath.Join(origin, name)
		destPath := filepath.Join(dest, name)

		if _, exists := destMap[name]; !exists {
			// File is new in origin
			fileChanged, err := copyItem(originPath, destPath, file.IsDir(), ignores)
			if err != nil {
				return false, err
			}
			changed = changed || fileChanged
		} else {
			// File exists in both, check for changes
			fileChanged, err := diffRecursive(originPath, destPath, dmp, ignores)
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
			err := RemoveDirectory(filepath.Join(dest, name))
			if err != nil {
				return false, err
			}
			changed = true
		}
	}

	return changed, nil
}

// diffFiles compares the contents of two files.
func diffFiles(origin, dest string, dmp *diffmatchpatch.DiffMatchPatch, ignores []string) (bool, error) {
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
		return copyItem(origin, dest, false, ignores)
	}
	return false, nil
}

// copyItem copies a file or directory from origin to dest.
func copyItem(origin, dest string, isDir bool, ignores []string) (bool, error) {
	if isDir {
		err := CopyDirectory(origin, dest, ignores)
		return true, err
	}
	err := CopyFile(origin, dest, ignores)
	return true, err
}

// replaceItem removes the destination item and replaces it with the origin item.
func replaceItem(origin, dest string, isDir bool, ignores []string) (bool, error) {
	err := RemoveDirectory(dest)
	if err != nil {
		return false, err
	}
	return copyItem(origin, dest, isDir, ignores)
}
