package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	changedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	unchangedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	alertStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	boldStyle      = lipgloss.NewStyle().Bold(true)

	// Security risk level styles
	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	highStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	mediumStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	lowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("118"))

	// Security scan styles
	scanningStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	securityPassStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
)

// GetCriticalStyle returns the critical risk level style
func GetCriticalStyle() lipgloss.Style {
	return criticalStyle
}

// GetHighStyle returns the high risk level style
func GetHighStyle() lipgloss.Style {
	return highStyle
}

// GetMediumStyle returns the medium risk level style
func GetMediumStyle() lipgloss.Style {
	return mediumStyle
}

// GetLowStyle returns the low risk level style
func GetLowStyle() lipgloss.Style {
	return lowStyle
}
