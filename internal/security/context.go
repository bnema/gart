package security

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bnema/gart/internal/system"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Choice option styles
	skipChoiceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true) // Orange
	proceedChoiceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red
	abortChoiceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)  // Green
	reviewChoiceStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)  // Blue
	openChoiceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true) // Magenta
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
		fmt.Printf("  [%s]pen in editor\n", openChoiceStyle.Render("o"))
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
		case "o", "open":
			proceed, skipAll, err := sc.openInEditor(report)
			return proceed, skipAll, err
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
			fmt.Print("\n[s]kip this file, [i]nclude anyway, [o]pen in editor, [a]bort: ")
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
			case "o", "open":
				editor := system.GetEditor()
				fmt.Printf("Opening %s in %s...\n", filepath.Base(result.FilePath), editor)
				
				cmd := exec.Command(editor, result.FilePath)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				
				if err := cmd.Run(); err != nil {
					fmt.Printf("Error opening file in editor: %v\n", err)
				}
				// Continue the loop to ask again what to do with this file
				continue
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

// openInEditor opens files with security findings in the user's preferred editor
func (sc *SecurityContext) openInEditor(report *ScanReport) (bool, bool, error) {
	editor := system.GetEditor()
	reader := bufio.NewReader(os.Stdin)
	
	// Get list of files with findings
	var filesWithFindings []string
	for _, result := range report.Results {
		if len(result.Findings) > 0 {
			filesWithFindings = append(filesWithFindings, result.FilePath)
		}
	}
	
	if len(filesWithFindings) == 0 {
		fmt.Println("No files with security findings to open.")
		return true, false, nil
	}
	
	// Show summary before opening files
	fmt.Printf("\nOpening %d file(s) with security findings in %s...\n", len(filesWithFindings), editor)
	for i, filePath := range filesWithFindings {
		fmt.Printf("  %d. %s\n", i+1, filePath)
	}
	fmt.Print("\nPress Enter to continue or Ctrl+C to cancel: ")
	_, err := reader.ReadString('\n')
	if err != nil {
		return false, false, fmt.Errorf("error reading input: %w", err)
	}
	
	// Open all files at once in the editor
	if err := sc.openMultipleFiles(editor, filesWithFindings); err != nil {
		fmt.Printf("Error opening files in editor: %v\n", err)
		
		// Ask if user wants to try opening files individually or abort
		for {
			fmt.Print("Try opening files one by one instead? [y/n]: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return false, false, fmt.Errorf("error reading input: %w", err)
			}
			
			choice := strings.ToLower(strings.TrimSpace(input))
			switch choice {
			case "y", "yes":
				if err := sc.openFilesSequentially(editor, filesWithFindings); err != nil {
					fmt.Printf("Error opening files sequentially: %v\n", err)
				}
				goto askUserChoice
			case "n", "no":
				return false, false, nil
			default:
				fmt.Printf("Invalid choice '%s'. Please enter 'y' or 'n'.\n", choice)
			}
		}
	}
	
askUserChoice:
	// After reviewing all files in editor, ask user what to do
	fmt.Print("\nAfter reviewing files in editor, choose your action:\n")
	fmt.Printf("  [%s]roceed with sync despite security issues\n", proceedChoiceStyle.Render("p"))
	fmt.Printf("  [%s]kip all (bypass security for all remaining dotfiles)\n", skipChoiceStyle.Render("s"))
	fmt.Printf("  [%s]bort sync\n", abortChoiceStyle.Render("a"))
	fmt.Print("Your choice: ")
	
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, false, fmt.Errorf("error reading input: %w", err)
		}
		
		choice := strings.ToLower(strings.TrimSpace(input))
		
		switch choice {
		case "p", "proceed":
			fmt.Println("Proceeding with sync despite security issues...")
			return true, false, nil
		case "s", "skip":
			fmt.Println("Skipping security prompts for all remaining dotfiles...")
			return true, true, nil
		case "a", "abort":
			fmt.Println("Aborting sync.")
			return false, false, nil
		default:
			fmt.Printf("Invalid choice '%s'. Please try again.\n", choice)
			fmt.Print("Your choice: ")
		}
	}
}

// openMultipleFiles opens all files at once in the editor using multiple arguments
func (sc *SecurityContext) openMultipleFiles(editor string, files []string) error {
	// Prepare command arguments
	args := make([]string, 0, len(files)+2)
	
	// Add editor-specific flags for better multi-file handling
	editorBase := filepath.Base(editor)
	switch strings.ToLower(editorBase) {
	case "code", "code-insiders":
		// VS Code: wait for editor to close and open in new window
		args = append(args, "--wait", "--new-window")
	case "subl", "sublime_text":
		// Sublime Text: wait for editor and open in new window
		args = append(args, "--wait", "--new-window")
	case "vim", "nvim", "neovim":
		// Vim/Neovim: open files in tabs
		args = append(args, "-p")
	case "emacs":
		// Emacs: no special flags needed, supports multiple files natively
	case "nano":
		// Nano doesn't support multiple files well, but we'll try anyway
		// The sequential fallback will handle this better if it fails
	default:
		// For unknown editors, try without special flags
	}
	
	// Add all file paths
	args = append(args, files...)
	
	fmt.Printf("Executing: %s %s\n", editor, strings.Join(args, " "))
	
	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// openFilesSequentially opens files one by one as a fallback
func (sc *SecurityContext) openFilesSequentially(editor string, files []string) error {
	reader := bufio.NewReader(os.Stdin)
	
	for i, filePath := range files {
		fmt.Printf("Opening file %d/%d: %s\n", i+1, len(files), filepath.Base(filePath))
		
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error opening file in editor: %v\n", err)
			
			// Ask if user wants to continue with next file or abort
			for {
				fmt.Print("Continue with next file? [y/n]: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("error reading input: %w", err)
				}
				
				choice := strings.ToLower(strings.TrimSpace(input))
				switch choice {
				case "y", "yes":
					goto nextFile
				case "n", "no":
					return fmt.Errorf("user chose to stop opening files")
				default:
					fmt.Printf("Invalid choice '%s'. Please enter 'y' or 'n'.\n", choice)
				}
			}
		}
		
	nextFile:
	}
	
	return nil
}

