package deploy

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	deploySummaryFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#334155")).
				Foreground(lipgloss.Color("#D1D5DB")).
				Padding(1, 2).
				Width(88)
	deploySummaryTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#F8FAFC"))
	deploySummarySectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878")).Bold(true)
	deploySummaryLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	deploySummaryValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
)

func printDeploymentSummary(serviceName, serviceID, environment, environmentType, planID, instanceActionType, finalInstanceID string) {
	var body strings.Builder
	body.WriteString(deploySummaryTitleStyle.Render("Deployment submitted"))
	body.WriteString("\n\n")
	body.WriteString(deploySummarySectionStyle.Render("Service"))
	body.WriteString("\n")
	body.WriteString(deploySummaryRow("Name", serviceName))
	body.WriteString(deploySummaryRow("ID", serviceID))
	body.WriteString(deploySummaryRow("Environment", fmt.Sprintf("%s (%s)", environment, environmentType)))
	body.WriteString(deploySummaryRow("Plan ID", planID))

	if finalInstanceID != "" {
		body.WriteString("\n")
		body.WriteString(deploySummarySectionStyle.Render("Instance"))
		body.WriteString("\n")
		body.WriteString(deploySummaryRow("Action", instanceActionType))
		body.WriteString(deploySummaryRow("ID", finalInstanceID))
	}

	fmt.Println()
	fmt.Println(deploySummaryFrameStyle.Render(strings.TrimRight(body.String(), "\n")))
	fmt.Println()
}

func deploySummaryRow(label, value string) string {
	return fmt.Sprintf("  %s  %s\n", deploySummaryLabelStyle.Width(12).Render(label+":"), deploySummaryValueStyle.Render(value))
}
