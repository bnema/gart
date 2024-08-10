package system

import "os"

func RemoveDirectory(path string) error {
	// Remove the directory
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}
