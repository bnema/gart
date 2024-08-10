package system

import (
	"os"
)

func GetStoragePath(name string) (string, error) {

	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Create the storage path
	storagePath := homeDir + "/.storage/" + name + ".toml"

	return storagePath, nil
}
