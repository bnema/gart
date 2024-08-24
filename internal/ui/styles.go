package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	noStyle        = lipgloss.NewStyle()
	helpStyle      = blurredStyle.Copy()
	focusedButton  = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton  = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	changedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	unchangedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	alertStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
)
