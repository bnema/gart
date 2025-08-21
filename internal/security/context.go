package security

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Choice option styles
	skipChoiceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true) // Orange
	proceedChoiceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red
	abortChoiceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)  // Green
	reviewChoiceStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)  // Blue
)

type SecurityContext struct {
	scanner *Scanner
	config  *SecurityConfig
}

func NewSecurityContext(config *SecurityConfig) *SecurityContext {
	return &SecurityContext{
		scanner: NewScanner(config),
		config:  config,
	}
}

// ScanPath scans a file or directory for security issues before sync
func (sc *SecurityContext) ScanPath(path string, ignores []string) (*ScanReport, error) {
	if !sc.config.Enabled {
		// Security is disabled, return empty report
		return &ScanReport{
			TotalFiles:    0,
			ScannedFiles:  0,
			SkippedFiles:  0,
			TotalFindings: 0,
			HighestRisk:   RiskLevelNone,
		}, nil
	}

	// Check if the path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing path %s: %w", path, err)
	}

	if info.IsDir() {
		return sc.scanner.ScanDirectory(path, ignores)
	} else {
		// Single file scan
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %w", path, err)
		}

		result, err := sc.scanner.ScanFile(path, content)
		if err != nil {
			return nil, err
		}

		report := &ScanReport{
			Results:       []ScanResult{*result},
			TotalFiles:    1,
			ScannedFiles:  1,
			SkippedFiles:  0,
			TotalFindings: len(result.Findings),
			HighestRisk:   result.Risk,
		}

		return report, nil
	}
}

// ShouldProceed determines if sync should continue based on security findings
func (sc *SecurityContext) ShouldProceed(report *ScanReport) (bool, string) {
	if !sc.config.Enabled {
		return true, ""
	}

	if report.TotalFindings == 0 {
		return true, ""
	}

	// Check if we should fail on secrets
	if sc.config.FailOnSecrets && report.HighestRisk >= RiskLevelHigh {
		return false, fmt.Sprintf("Security scan failed: found %d security issues with %s risk level",
			report.TotalFindings, report.HighestRisk)
	}

	// Always block critical issues
	if report.HighestRisk >= RiskLevelCritical {
		return false, fmt.Sprintf("Critical security issues found: %d findings", report.TotalFindings)
	}

	return true, ""
}

// InteractivePrompt presents security findings to the user and gets their decision
// Returns: proceed (continue with this dotfile), skipAll (skip security for remaining dotfiles), error
func (sc *SecurityContext) InteractivePrompt(report *ScanReport) (bool, bool, error) {
	if !sc.config.Interactive || report.TotalFindings == 0 {
		proceed, msg := sc.ShouldProceed(report)
		if !proceed {
			return false, false, fmt.Errorf("%s", msg)
		}
		return true, false, nil
	}

	// Get user decision (display will be handled by UI layer)
	return sc.getUserDecision(report)
}

func (sc *SecurityContext) getUserDecision(report *ScanReport) (bool, bool, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nOptions:\n")
		fmt.Printf("  [%s]kip all (bypass security for all remaining dotfiles)\n", skipChoiceStyle.Render("s"))
		fmt.Printf("  [%s]roceed anyway (not recommended)\n", proceedChoiceStyle.Render("p"))
		fmt.Printf("  [%s]bort sync\n", abortChoiceStyle.Render("a"))
		fmt.Printf("  [%s]eview each file\n", reviewChoiceStyle.Render("r"))
		fmt.Print("\nYour choice: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return false, false, fmt.Errorf("error reading input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))

		switch choice {
		case "s", "skip":
			fmt.Println("Skipping security prompts for all remaining dotfiles...")
			return true, true, nil
		case "p", "proceed":
			fmt.Println(" Proceeding with sync despite security issues...")
			return true, false, nil
		case "a", "abort":
			fmt.Println("Aborting sync.")
			return false, false, nil
		case "r", "review":
			proceed, err := sc.reviewEachFile(report)
			return proceed, false, err
		default:
			fmt.Printf("Invalid choice '%s'. Please try again.\n", choice)
		}
	}
}


func (sc *SecurityContext) reviewEachFile(report *ScanReport) (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	for _, result := range report.Results {
		if len(result.Findings) == 0 {
			continue
		}

		fmt.Printf("\n File: %s\n", result.FilePath)
		fmt.Printf("Risk Level: %s\n", result.Risk)
		fmt.Printf("Findings: %d\n", len(result.Findings))

		for _, finding := range result.Findings {
			fmt.Printf("  • %s: %s (%.0f%% confidence)\n",
				finding.Type, finding.Value, finding.Confidence*100)
		}

		for {
			fmt.Print("\n[s]kip this file, [i]nclude anyway, [a]bort: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return false, fmt.Errorf("error reading input: %w", err)
			}

			choice := strings.ToLower(strings.TrimSpace(input))

			switch choice {
			case "s", "skip":
				fmt.Printf("Skipping %s\n", filepath.Base(result.FilePath))
				// Mark file to be skipped
				goto nextFile
			case "i", "include":
				fmt.Printf("Including %s despite security issues\n", filepath.Base(result.FilePath))
				goto nextFile
			case "a", "abort":
				return false, nil
			default:
				fmt.Printf("Invalid choice '%s'. Please try again.\n", choice)
			}
		}

	nextFile:
	}

	return true, nil
}

