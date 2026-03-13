package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// skill represents an installed Omnistrate agent skill.
type skill struct {
	Name        string
	Description string
	Dir         string   // absolute path to skill directory
	Files       []string // filenames (SKILL.md, reference docs)
}

// discoverSkills looks for installed skills in both local and global .claude/skills directories.
func discoverSkills(cwd string) []skill {
	homeDir, _ := os.UserHomeDir()

	searchDirs := []string{
		filepath.Join(cwd, ".claude", "skills"),
	}
	if homeDir != "" {
		searchDirs = append(searchDirs, filepath.Join(homeDir, ".claude", "skills"))
	}

	seen := map[string]bool{}
	var skills []skill

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			// Skip if already seen (local takes priority over global)
			if seen[e.Name()] {
				continue
			}

			skillDir := filepath.Join(dir, e.Name())
			s := parseSkill(skillDir)
			if s != nil {
				seen[e.Name()] = true
				skills = append(skills, *s)
			}
		}
	}

	return skills
}

// parseSkill reads a skill directory and extracts metadata from SKILL.md frontmatter.
func parseSkill(dir string) *skill {
	skillPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	s := &skill{
		Dir:  dir,
		Name: filepath.Base(dir),
	}

	// Parse YAML frontmatter (between --- lines)
	content := string(data)
	if strings.HasPrefix(content, "---\n") {
		endIdx := strings.Index(content[4:], "\n---")
		if endIdx >= 0 {
			frontmatter := content[4 : 4+endIdx]
			for _, line := range strings.Split(frontmatter, "\n") {
				if strings.HasPrefix(line, "name:") {
					s.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				} else if strings.HasPrefix(line, "description:") {
					s.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				}
			}
		}
	}

	// Collect all files in the skill directory
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				s.Files = append(s.Files, e.Name())
			}
		}
	}

	return s
}

// loadSkillContent reads all markdown files from a skill directory
// and returns concatenated content, respecting a size cap.
func loadSkillContent(s skill, maxBytes int) string {
	var sb strings.Builder
	remaining := maxBytes

	for _, filename := range s.Files {
		data, err := os.ReadFile(filepath.Join(s.Dir, filename))
		if err != nil {
			continue
		}
		content := string(data)

		// Strip frontmatter from SKILL.md
		if filename == "SKILL.md" && strings.HasPrefix(content, "---\n") {
			endIdx := strings.Index(content[4:], "\n---")
			if endIdx >= 0 {
				content = strings.TrimSpace(content[4+endIdx+4:])
			}
		}

		if len(content) > remaining {
			content = content[:remaining] + "\n... (truncated)"
			sb.WriteString(fmt.Sprintf("\n### %s\n", filename))
			sb.WriteString(content)
			break
		}

		sb.WriteString(fmt.Sprintf("\n### %s\n", filename))
		sb.WriteString(content)
		sb.WriteByte('\n')
		remaining -= len(content)
	}

	return sb.String()
}

// promptSkillSelection shows available skills and lets user select one interactively.
// Returns nil if user skips (selects 0) or no skills available.
func promptSkillSelection(skills []skill) *skill {
	if len(skills) == 0 {
		fmt.Println("📋 No skills installed. Run 'omnistrate-ctl agent init' to install skills.")
		return nil
	}

	fmt.Println("📋 Available skills:")
	fmt.Println()
	for i, s := range skills {
		fmt.Printf("  [%d] %s\n", i+1, s.Name)
		if s.Description != "" {
			// Truncate description for display
			desc := s.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			fmt.Printf("      %s\n", desc)
		}
	}
	fmt.Println()
	fmt.Printf("  [0] No skill (general chat)\n")
	fmt.Println()
	fmt.Print("Select a skill: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}
	response = strings.TrimSpace(response)

	// Parse selection
	var selection int
	if _, err := fmt.Sscanf(response, "%d", &selection); err != nil {
		return nil
	}

	if selection < 1 || selection > len(skills) {
		return nil
	}

	return &skills[selection-1]
}
