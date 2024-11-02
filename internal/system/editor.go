package system

import (
	"os"
	"os/exec"
)


func GetEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Try nano if no editor is set
	if _, err := exec.LookPath("nano"); err == nil {
		return "nano"
	}
	return "vi" // Fallback to vi if no editor is set
}
