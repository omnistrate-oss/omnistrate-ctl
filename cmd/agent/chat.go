package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/mcpbridge"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/provider"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/tui"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	// Register all providers via their init() functions.
	_ "github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/provider"
)

// Spec file names in priority order (same as cmd/build and cmd/deploy).
var specFileNames = []string{
	"omnistrate-compose.yaml",
	"omnistrate-compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
	"compose.yaml",
	"compose.yml",
	"spec.yaml",
	"spec.yml",
}

var chatCmd = &cobra.Command{
	Use:   "chat [provider]",
	Short: "Start an interactive AI agent chat with Omnistrate tools",
	Long: `Start an interactive TUI chat session with an AI provider that has access
to Omnistrate tools via the built-in MCP server.

Supported providers: claude, chatgpt, copilot

The AI agent can help you:
- Create and edit omnistrate-compose.yaml spec files
- Build and deploy services
- Manage instances and subscriptions
- Debug deployment issues`,
	Args: cobra.ExactArgs(1),
	RunE: runChat,
	Example: `  omnistrate-ctl agent chat claude
  omnistrate-ctl agent chat chatgpt
  omnistrate-ctl agent chat copilot`,
	SilenceUsage: true,
}

func runChat(cmd *cobra.Command, args []string) error {
	providerName := strings.ToLower(args[0])

	// Step 1: Ensure the user is logged into Omnistrate
	fmt.Println("Checking Omnistrate authentication...")
	_, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("omnistrate login required: %w", err)
	}
	fmt.Println("✅ Authenticated with Omnistrate")

	// Step 2: Check provider
	p, err := provider.Get(providerName)
	if err != nil {
		return fmt.Errorf("unknown provider %q. Available: %s", providerName, strings.Join(provider.Names(), ", "))
	}
	ok, hint := p.IsConfigured()
	if !ok {
		return fmt.Errorf("provider %q is not configured: %s", providerName, hint)
	}

	// Step 3: Discover current directory and spec file
	cwd, _ := os.Getwd()
	specFile, specContent := detectSpec(cwd)

	// Step 4: Auto-init skills if none are installed
	skills := discoverSkills(cwd)
	if len(skills) == 0 {
		fmt.Println()
		fmt.Println("🔧 No skills found. Running agent init to install Omnistrate skills...")
		fmt.Println()
		if initErr := runInitSilent(cwd); initErr != nil {
			fmt.Printf("⚠️  Skills init failed: %v (continuing without skills)\n", initErr)
		} else {
			skills = discoverSkills(cwd)
			fmt.Printf("✅ Installed %d skill(s)\n", len(skills))
		}
	}

	// Step 5: Let user select a skill
	fmt.Println()
	selectedSkill := promptSkillSelection(skills)
	if selectedSkill != nil {
		fmt.Printf("🎯 Using skill: %s\n", selectedSkill.Name)
	} else {
		fmt.Println("💬 Starting general chat (no skill selected)")
	}

	// Step 6: Show directory overview
	fmt.Println()
	fmt.Printf("📂 Working directory: %s\n", cwd)
	listDirectory(cwd)
	if specFile != "" {
		fmt.Printf("📄 Spec file found: %s\n", specFile)
	} else {
		fmt.Println("📄 No spec file found (will create one if needed)")
	}
	fmt.Println()

	// Step 7: Build system prompt with spec context and skill
	systemPrompt := buildSystemPrompt(cwd, specFile, specContent, selectedSkill)

	// Step 8: Create MCP bridge and run TUI
	bridge := mcpbridge.New()
	model := tui.New(p, bridge, systemPrompt)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	bridge.Close()
	return nil
}

// detectSpec looks for a spec file in cwd using the standard priority order.
func detectSpec(cwd string) (name string, content string) {
	for _, candidate := range specFileNames {
		path := filepath.Join(cwd, candidate)
		data, err := os.ReadFile(path)
		if err == nil {
			return candidate, string(data)
		}
	}
	return "", ""
}

// listDirectory prints visible files in the directory.
func listDirectory(cwd string) {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return
	}
	fmt.Println("  Files:")
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		indicator := "📁"
		if !e.IsDir() {
			indicator = "  "
		}
		fmt.Printf("    %s %s\n", indicator, e.Name())
	}
}

func buildSystemPrompt(cwd, specFile, specContent string, selectedSkill *skill) string {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant integrated into omnistrate-ctl, a CLI for managing Omnistrate SaaS deployments.\n\n")

	// Include skill instructions if selected
	if selectedSkill != nil {
		fmt.Fprintf(&sb, "## Active Skill: %s\n", selectedSkill.Name)
		if selectedSkill.Description != "" {
			sb.WriteString(selectedSkill.Description + "\n")
		}
		// Load skill content (cap at 2K to leave room for spec + conversation within token limits)
		skillContent := loadSkillContent(*selectedSkill, 2000)
		if skillContent != "" {
			sb.WriteString(skillContent)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Context\n")
	fmt.Fprintf(&sb, "- Working directory: %s\n", cwd)

	// List files in cwd
	entries, err := os.ReadDir(cwd)
	if err == nil {
		sb.WriteString("- Files in current directory:\n")
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), ".") {
				kind := "file"
				if e.IsDir() {
					kind = "dir"
				}
				fmt.Fprintf(&sb, "  - %s (%s)\n", e.Name(), kind)
			}
		}
	}

	// Include spec file content
	if specFile != "" && specContent != "" {
		fmt.Fprintf(&sb, "\n## Current Spec File: %s\n", specFile)
		sb.WriteString("```yaml\n")
		// Cap spec content to avoid blowing up token limits
		if len(specContent) > 1500 {
			sb.WriteString(specContent[:1500])
			sb.WriteString("\n... (truncated)\n")
		} else {
			sb.WriteString(specContent)
		}
		sb.WriteString("\n```\n")
		sb.WriteString("This is the user's current service spec. Refer to it when discussing builds, deploys, or changes.\n")
	} else {
		sb.WriteString("\n## No Spec File Found\n")
		sb.WriteString("No omnistrate-compose.yaml or similar spec file found in the current directory.\n")
		sb.WriteString("Offer to create one if the user wants to build/deploy a service.\n")
	}

	sb.WriteString("\n## Capabilities\n")
	sb.WriteString("You have access to Omnistrate MCP tools for service management, instance ops, builds, and deploys.\n\n")

	sb.WriteString("## File Editing\n")
	sb.WriteString("When creating or editing files, output code blocks with the filename:\n")
	sb.WriteString("```yaml omnistrate-compose.yaml\nservices:\n  ...\n```\n\n")

	sb.WriteString("## Omnistrate Compose Format\n")
	sb.WriteString("Omnistrate services use Docker Compose YAML with x-omnistrate-* extensions.\n")
	sb.WriteString("Key: x-omnistrate-service-plan, x-omnistrate-api-params, x-omnistrate-compute, x-omnistrate-storage.\n")

	return sb.String()
}
