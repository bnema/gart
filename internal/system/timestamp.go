package system

import "time"

// GetTimestamp returns the current timestamp
func GetTimestamp() string {
    return time.Now().Format(time.RFC3339)
}
