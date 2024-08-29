package system

import "os"

// GetHostname returns the hostname of the system
func GetHostname() (string, error) {
	return os.Hostname()
}
