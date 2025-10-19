package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	agentInstructionsRepo   = "https://github.com/omnistrate-oss/agent-instructions"
	omnistrateSectionHeader = "## Omnistrate Agent Instructions\n\n"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Claude Code skills and agent instructions for Omnistrate",
	Long: `Installs Claude Code skills and agent instructions for Omnistrate.
This command will:
1. Clone the agent-instructions repository
2. Copy skills to .claude/skills/ directory
3. Merge AGENTS.md and CLAUDE.md into your project`,
	RunE:         runInstall,
	SilenceUsage: true,
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create temp directory for cloning
	tmpDir := filepath.Join(os.TempDir(), "omnistrate-agent-instructions")

	// Show user what will happen
	fmt.Println("Omnistrate Agent Installation")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("This will setup the Omnistrate agent and:")
	fmt.Printf("1. Copy skills to %s\n", filepath.Join(cwd, ".claude", "skills"))
	fmt.Printf("2. Merge AGENTS.md content into %s\n", filepath.Join(cwd, "AGENTS.md"))
	fmt.Printf("3. Merge CLAUDE.md content into %s\n", filepath.Join(cwd, "CLAUDE.md"))
	fmt.Println()
	fmt.Print("Do you want to continue? (yes/no): ")

	// Read user confirmation
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Println("Installation cancelled.")
		return nil
	}

	fmt.Println()
	fmt.Println("Installing Claude Skills...")
	fmt.Println()

	// Step 1: Clone repository
	fmt.Println("Cloning agent-instructions repository...")

	// Clean up old temp directory if it exists
	if _, err := os.Stat(tmpDir); err == nil {
		if err := os.RemoveAll(tmpDir); err != nil {
			return fmt.Errorf("failed to clean up old temp directory: %w", err)
		}
	}

	cloneCmd := exec.Command("git", "clone", agentInstructionsRepo, tmpDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	defer func() {
		// Clean up temp directory
		os.RemoveAll(tmpDir)
	}()

	// Read README to list skills
	readmePath := filepath.Join(tmpDir, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not read README.md: %v\n", err)
	} else {
		// Extract skills from README
		fmt.Println("\n📚 Installing the following skills:")
		printSkillsFromReadme(string(readmeContent))
		fmt.Println()
	}

	// Step 2: Copy skills directory
	fmt.Println("Copying skills to .claude/skills/...")

	destSkillsDir := filepath.Join(cwd, ".claude", "skills")
	srcSkillsDir := filepath.Join(tmpDir, "skills")

	// Create .claude/skills directory if it doesn't exist
	if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude/skills directory: %w", err)
	}

	// Get list of skill directories from source to remove old versions
	srcSkills, err := os.ReadDir(srcSkillsDir)
	if err != nil {
		return fmt.Errorf("failed to read source skills directory: %w", err)
	}

	// Remove existing skill directories with the same name
	for _, skill := range srcSkills {
		if skill.IsDir() {
			destSkillPath := filepath.Join(destSkillsDir, skill.Name())
			if _, err := os.Stat(destSkillPath); err == nil {
				fmt.Printf("  🗑️  Removing existing skill: %s\n", skill.Name())
				if err := os.RemoveAll(destSkillPath); err != nil {
					return fmt.Errorf("failed to remove existing skill %s: %w", skill.Name(), err)
				}
			}
		}
	}

	// Copy skills directory
	if err := copyDir(srcSkillsDir, destSkillsDir); err != nil {
		return fmt.Errorf("failed to copy skills directory: %w", err)
	}

	fmt.Println("✅ Skills copied successfully")

	// Step 3: Merge AGENTS.md
	fmt.Println("Merging AGENTS.md...")
	agentsSrcPath := filepath.Join(tmpDir, "AGENTS.md")
	agentsDestPath := filepath.Join(cwd, "AGENTS.md")

	if err := mergeMarkdownFile(agentsSrcPath, agentsDestPath); err != nil {
		return fmt.Errorf("failed to merge AGENTS.md: %w", err)
	}

	fmt.Println("✅ AGENTS.md updated successfully")

	// Step 4: Merge CLAUDE.md
	fmt.Println("\n📝 Merging CLAUDE.md...")
	claudeSrcPath := filepath.Join(tmpDir, "CLAUDE.md")
	claudeDestPath := filepath.Join(cwd, "CLAUDE.md")

	if err := mergeMarkdownFile(claudeSrcPath, claudeDestPath); err != nil {
		return fmt.Errorf("failed to merge CLAUDE.md: %w", err)
	}

	fmt.Println("✅ CLAUDE.md updated successfully")

	fmt.Println("Installation complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Review .claude/skills/ to see installed skills")
	fmt.Println("  - Check AGENTS.md for agent instructions")
	fmt.Println("  - Check CLAUDE.md for Claude Code skill configuration")
	fmt.Println()

	return nil
}

func printSkillsFromReadme(readme string) {
	// Look for the skills section and extract skill names and descriptions
	lines := strings.Split(readme, "\n")
	inSkillsSection := false

	for i, line := range lines {
		if strings.Contains(line, "### Skills") || strings.Contains(line, "## Skills") {
			inSkillsSection = true
			continue
		}

		if inSkillsSection {
			// Stop at next major section
			if strings.HasPrefix(line, "## ") && !strings.Contains(line, "skills/") {
				break
			}

			// Look for skill headers (starts with #### and contains skills/)
			if strings.HasPrefix(line, "####") && strings.Contains(line, "skills/") {
				// Extract skill name
				skillLine := strings.TrimPrefix(line, "####")
				skillLine = strings.TrimSpace(skillLine)

				// Parse [**skills/name/**] or **skills/name/** format
				if strings.Contains(skillLine, "](") {
					// Markdown link format
					parts := strings.Split(skillLine, "]")
					if len(parts) > 0 {
						skillName := strings.TrimPrefix(parts[0], "[")
						skillName = strings.Trim(skillName, "*")
						fmt.Printf("  • %s\n", skillName)
					}
				} else {
					// Direct format
					skillName := strings.Trim(skillLine, "*")
					fmt.Printf("  • %s\n", skillName)
				}

				// Print description from next line if available
				if i+1 < len(lines) && lines[i+1] != "" && !strings.HasPrefix(lines[i+1], "#") {
					desc := strings.TrimSpace(lines[i+1])
					if desc != "" {
						fmt.Printf("    %s\n", desc)
					}
				}
			}
		}
	}
}

func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func mergeMarkdownFile(srcPath, destPath string) error {
	// Read source content
	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Check if destination file exists
	var destContent []byte
	if _, err := os.Stat(destPath); err == nil {
		// File exists, read it
		destContent, err = os.ReadFile(destPath)
		if err != nil {
			return fmt.Errorf("failed to read destination file: %w", err)
		}

		// Check if Omnistrate section already exists
		if strings.Contains(string(destContent), omnistrateSectionHeader) {
			// Replace existing Omnistrate section
			destContent = []byte(replaceOmnistrateSection(string(destContent), string(srcContent)))
		} else {
			// Append Omnistrate section
			destContent = append(destContent, []byte("\n\n"+omnistrateSectionHeader)...)
			destContent = append(destContent, srcContent...)
		}
	} else if os.IsNotExist(err) {
		// File doesn't exist, create new with section header
		destContent = []byte(omnistrateSectionHeader)
		destContent = append(destContent, srcContent...)
	} else {
		return fmt.Errorf("failed to check destination file: %w", err)
	}

	// Write merged content
	if err := os.WriteFile(destPath, destContent, 0600); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

func replaceOmnistrateSection(destContent, srcContent string) string {
	// Find the start of the Omnistrate section
	startIdx := strings.Index(destContent, omnistrateSectionHeader)
	if startIdx == -1 {
		// Section not found, shouldn't happen but handle gracefully
		return destContent + "\n\n" + omnistrateSectionHeader + srcContent
	}

	// Find the end of the Omnistrate section (next ## header or end of file)
	searchStart := startIdx + len(omnistrateSectionHeader)
	endIdx := len(destContent)

	// Look for next ## header after the Omnistrate section
	remainingContent := destContent[searchStart:]
	if nextHeaderIdx := strings.Index(remainingContent, "\n## "); nextHeaderIdx != -1 {
		endIdx = searchStart + nextHeaderIdx
	}

	// Replace the section
	before := destContent[:startIdx]
	after := ""
	if endIdx < len(destContent) {
		after = destContent[endIdx:]
	}

	return before + omnistrateSectionHeader + srcContent + after
}
