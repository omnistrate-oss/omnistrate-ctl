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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Claude Code skills and agent instructions for Omnistrate",
	Long: `Initializes Claude Code skills and agent instructions for Omnistrate.
This command will:
1. Clone the agent-instructions repository (or use local directory)
2. Copy skills to .claude/skills/ directory
3. Merge AGENTS.md and CLAUDE.md into your project`,
	RunE:         runInit,
	SilenceUsage: true,
}

func init() {
	initCmd.Flags().String("instruction-source", "", "Path to local agent-instructions directory (default: clones from GitHub)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get instruction source flag
	instructionSource, err := cmd.Flags().GetString("instruction-source")
	if err != nil {
		return fmt.Errorf("failed to get instruction-source flag: %w", err)
	}

	// Determine source directory
	var sourceDir string
	var cleanupTempDir bool

	if instructionSource != "" {
		// Use local directory
		absPath, err := filepath.Abs(instructionSource)
		if err != nil {
			return fmt.Errorf("failed to resolve instruction-source path: %w", err)
		}

		// Verify the directory exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("instruction-source directory does not exist: %s", absPath)
		}

		sourceDir = absPath
		cleanupTempDir = false
	} else {
		// Clone from GitHub
		sourceDir = filepath.Join(os.TempDir(), "omnistrate-agent-instructions")
		cleanupTempDir = true
	}

	// Get home directory for global skills
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Show user what will happen
	fmt.Println("Omnistrate Agent Initialization")
	fmt.Println("==============================")
	fmt.Println()
	if instructionSource != "" {
		fmt.Printf("Using local directory: %s\n", sourceDir)
		fmt.Println()
	}
	fmt.Println("This will setup the Omnistrate agent and:")
	fmt.Printf("1. Copy skills to %s\n", filepath.Join(cwd, ".claude", "skills"))
	fmt.Printf("2. Copy skills to %s (global)\n", filepath.Join(homeDir, ".claude", "skills"))
	fmt.Printf("3. Merge AGENTS.md content into %s\n", filepath.Join(cwd, "AGENTS.md"))
	fmt.Printf("4. Merge AGENTS.md content into %s\n", filepath.Join(cwd, "GEMINI.md"))
	fmt.Printf("5. Merge CLAUDE.md content into %s\n", filepath.Join(cwd, "CLAUDE.md"))
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
		fmt.Println("Initialization cancelled.")
		return nil
	}

	fmt.Println()
	fmt.Println("Initializing Claude Skills...")
	fmt.Println()

	// Step 1: Prepare source directory
	if cleanupTempDir {
		fmt.Println("Cloning agent-instructions repository...")

		// Clean up old temp directory if it exists
		if _, err := os.Stat(sourceDir); err == nil {
			if err := os.RemoveAll(sourceDir); err != nil {
				return fmt.Errorf("failed to clean up old temp directory: %w", err)
			}
		}

		cloneCmd := exec.Command("git", "clone", agentInstructionsRepo, sourceDir)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		defer func() {
			// Clean up temp directory
			os.RemoveAll(sourceDir)
		}()
	} else {
		fmt.Printf("Using local directory: %s\n", sourceDir)
	}

	// Read README to list skills
	readmePath := filepath.Join(sourceDir, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not read README.md: %v\n", err)
	} else {
		// Extract skills from README
		fmt.Println("Initializing the following skills:")
		printSkillsFromReadme(string(readmeContent))
		fmt.Println()
	}

	// Step 2: Copy skills directory to both local and global locations
	srcSkillsDir := filepath.Join(sourceDir, "skills")

	// Get list of skill directories from source to remove old versions
	srcSkills, err := os.ReadDir(srcSkillsDir)
	if err != nil {
		return fmt.Errorf("failed to read source skills directory: %w", err)
	}

	// Helper function to install skills to a destination
	installSkills := func(destSkillsDir string, location string) error {
		fmt.Printf("Copying skills to %s...\n", location)

		// Create .claude/skills directory if it doesn't exist
		if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", location, err)
		}

		// Remove existing skill directories with the same name
		for _, skill := range srcSkills {
			if skill.IsDir() {
				destSkillPath := filepath.Join(destSkillsDir, skill.Name())
				if _, err := os.Stat(destSkillPath); err == nil {
					fmt.Printf("   ️  Removing existing skill: %s\n", skill.Name())
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

		fmt.Printf("✅ Skills copied successfully to %s\n", location)
		return nil
	}

	// Install to local .claude/skills
	localSkillsDir := filepath.Join(cwd, ".claude", "skills")
	if err := installSkills(localSkillsDir, ".claude/skills"); err != nil {
		return err
	}

	fmt.Println()

	// Install to global ~/.claude/skills
	globalSkillsDir := filepath.Join(homeDir, ".claude", "skills")
	if err := installSkills(globalSkillsDir, "~/.claude/skills"); err != nil {
		return err
	}

	// Step 3: Merge AGENTS.md
	fmt.Println("Merging AGENTS.md...")
	agentsSrcPath := filepath.Join(sourceDir, "AGENTS.md")
	agentsDestPath := filepath.Join(cwd, "AGENTS.md")

	// Read, update skill paths, and merge
	agentsContent, err := os.ReadFile(agentsSrcPath)
	if err != nil {
		return fmt.Errorf("failed to read AGENTS.md: %w", err)
	}
	agentsContent = []byte(updateSkillPaths(string(agentsContent)))

	if err := mergeMarkdownFileWithContent(string(agentsContent), agentsDestPath); err != nil {
		return fmt.Errorf("failed to merge AGENTS.md: %w", err)
	}

	fmt.Println("✅ AGENTS.md updated successfully")

	// Step 4: Merge AGENTS.md content into GEMINI.md (same as AGENTS.md)
	fmt.Println("Merging AGENTS.md content into GEMINI.md...")
	geminiDestPath := filepath.Join(cwd, "GEMINI.md")

	if err := mergeMarkdownFileWithContent(string(agentsContent), geminiDestPath); err != nil {
		return fmt.Errorf("failed to merge GEMINI.md: %w", err)
	}

	fmt.Println("✅ GEMINI.md updated successfully")

	// Step 5: Merge CLAUDE.md
	fmt.Println("Merging CLAUDE.md...")
	claudeSrcPath := filepath.Join(sourceDir, "CLAUDE.md")
	claudeDestPath := filepath.Join(cwd, "CLAUDE.md")

	// Read, update skill paths, and merge
	claudeContent, err := os.ReadFile(claudeSrcPath)
	if err != nil {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}
	claudeContent = []byte(updateSkillPaths(string(claudeContent)))

	if err := mergeMarkdownFileWithContent(string(claudeContent), claudeDestPath); err != nil {
		return fmt.Errorf("failed to merge CLAUDE.md: %w", err)
	}

	fmt.Println("✅ CLAUDE.md updated successfully")

	fmt.Println("Initialization complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Review .claude/skills/ (local) and ~/.claude/skills/ (global) to see installed skills")
	fmt.Println("  - Check AGENTS.md and GEMINI.md for agent instructions")
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

func updateSkillPaths(content string) string {
	// Replace all instances of **Location**: `skills/ with **Location**: `.claude/skills/
	// This handles patterns like:
	// - **Location**: `skills/omnistrate-fde/`
	// - **Location**: `skills/omnistrate-sa/`
	// - **Location**: `skills/omnistrate-sre/`

	// Use regex to match and replace the pattern
	updated := strings.ReplaceAll(content, "**Location**: `skills/", "**Location**: `.claude/skills/")

	return updated
}

func mergeMarkdownFileWithContent(srcContent, destPath string) error {
	// Check if destination file exists
	var destContent []byte
	if _, err := os.Stat(destPath); err == nil {
		// File exists, read it
		destContent, err = os.ReadFile(destPath)
		if err != nil {
			return fmt.Errorf("failed to read destination file: %w", err)
		}

		// Update skill paths in existing content before merging
		existingContentStr := updateSkillPaths(string(destContent))

		// Check if Omnistrate section already exists
		if strings.Contains(existingContentStr, omnistrateSectionHeader) {
			// Replace existing Omnistrate section
			destContent = []byte(replaceOmnistrateSection(existingContentStr, srcContent))
		} else {
			// Append Omnistrate section
			destContent = []byte(existingContentStr + "\n\n" + omnistrateSectionHeader + srcContent)
		}
	} else if os.IsNotExist(err) {
		// File doesn't exist, create new with section header
		destContent = []byte(omnistrateSectionHeader + srcContent)
	} else {
		return fmt.Errorf("failed to check destination file: %w", err)
	}

	// Write merged content
	if err := os.WriteFile(destPath, destContent, 0600); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

func mergeMarkdownFile(srcPath, destPath string) error {
	// Read source content
	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	return mergeMarkdownFileWithContent(string(srcContent), destPath)
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
		// Ensure proper spacing before next section
		if !strings.HasPrefix(after, "\n\n") && strings.HasPrefix(after, "\n") {
			after = "\n" + after
		}
	}

	return before + omnistrateSectionHeader + srcContent + after
}
