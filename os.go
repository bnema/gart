package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func loadConfig(dotfiles map[string]string) {
	configPath := "config.toml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	for key, value := range config {
		if path, ok := value.(string); ok {
			dotfiles[key] = path
		}
	}
}

func saveConfig(dotfiles map[string]string) {
	data, err := toml.Marshal(dotfiles)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}

	err = os.WriteFile("config.toml", data, 0644)
	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func copyDirectory(src, dst string) error {
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
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Skip symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			// Copy regular files
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
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

func diffFiles(path1, path2 string) (bool, error) {
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

		info2, err := os.Stat(p2)
		if os.IsNotExist(err) {
			// The file exists in path1 but not in path2
			// Copy the file from path1 to path2
			if err := copyFile(p1, p2); err != nil {
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
				if err := copyFile(p1, p2); err != nil {
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
