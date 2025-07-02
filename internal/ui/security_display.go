package ui

import (
	"fmt"
	"strings"

	"github.com/bnema/gart/internal/security"
	"github.com/charmbracelet/lipgloss"
)

// DisplaySecurityReport displays a security report with proper styling
func DisplaySecurityReport(report *security.ScanReport) {
	fmt.Printf("\n%s\n", boldStyle.Render("󰒃 Security Scan Results:"))
	fmt.Printf("Scanned %d files, found %d security issues\n\n", report.ScannedFiles, report.TotalFindings)

	// Group findings by risk level
	riskGroups := make(map[security.RiskLevel][]security.Finding)
	for _, result := range report.Results {
		for _, finding := range result.Findings {
			riskGroups[finding.RiskLevel] = append(riskGroups[finding.RiskLevel], finding)
		}
	}

	// Display findings by risk level
	levels := []security.RiskLevel{
		security.RiskLevelCritical,
		security.RiskLevelHigh,
		security.RiskLevelMedium,
		security.RiskLevelLow,
	}

	for _, level := range levels {
		findings := riskGroups[level]
		if len(findings) == 0 {
			continue
		}

		icon := getRiskIcon(level)
		style := getRiskStyle(level)

		fmt.Printf("%s\n", style.Render(fmt.Sprintf("%s %s RISK (%d issues):", icon, level.String(), len(findings))))

		for _, finding := range findings {
			fmt.Printf("  %s %s\n", icon, finding.Location.FilePath)
			if finding.Value != "" {
				fmt.Printf("     └─ Found: %s\n", finding.Value)
			}
			fmt.Printf("     └─ Type: %s (confidence: %.0f%%)\n", finding.Type, finding.Confidence*100)
			if len(finding.Reasons) > 0 {
				fmt.Printf("     └─ Reason: %s\n", strings.Join(finding.Reasons, ", "))
			}
		}
		fmt.Println()
	}
}

func getRiskIcon(level security.RiskLevel) string {
	switch level {
	case security.RiskLevelCritical:
		return ""
	case security.RiskLevelHigh:
		return ""
	case security.RiskLevelMedium:
		return "󱈸"
	case security.RiskLevelLow:
		return ""
	default:
		return "•"
	}
}

func getRiskStyle(level security.RiskLevel) lipgloss.Style {
	switch level {
	case security.RiskLevelCritical:
		return criticalStyle
	case security.RiskLevelHigh:
		return highStyle
	case security.RiskLevelMedium:
		return mediumStyle
	case security.RiskLevelLow:
		return lowStyle
	default:
		return lipgloss.NewStyle()
	}
}
