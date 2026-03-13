package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	// Write SKILL.md with frontmatter
	skillContent := `---
name: Test Skill
description: A test skill for validation
---

# Test Skill

Instructions here.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "REFERENCE.md"), []byte("# Reference\nExtra info."), 0600))

	s := parseSkill(skillDir)
	require.NotNil(t, s)
	require.Equal(t, "Test Skill", s.Name)
	require.Equal(t, "A test skill for validation", s.Description)
	require.Contains(t, s.Files, "SKILL.md")
	require.Contains(t, s.Files, "REFERENCE.md")
}

func TestParseSkill_NoSkillFile(t *testing.T) {
	tmpDir := t.TempDir()
	s := parseSkill(tmpDir)
	require.Nil(t, s)
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create local .claude/skills with one skill
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: My Skill\ndescription: desc\n---\n# Skill"), 0600))

	skills := discoverSkills(tmpDir)
	// Should find at least the local one (may also find global ones)
	found := false
	for _, s := range skills {
		if s.Name == "My Skill" {
			found = true
			break
		}
	}
	require.True(t, found, "should discover the local skill")
}

func TestLoadSkillContent(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillMD := "---\nname: X\ndescription: Y\n---\n\n# Skill X\n\nInstructions for skill X."
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "REF.md"), []byte("# Reference\nMore details."), 0600))

	s := parseSkill(skillDir)
	require.NotNil(t, s)

	content := loadSkillContent(*s, 10000)
	require.Contains(t, content, "Skill X")
	require.Contains(t, content, "Instructions for skill X")
	require.Contains(t, content, "Reference")
	// Frontmatter should be stripped
	require.NotContains(t, content, "name: X")
}

func TestLoadSkillContent_Truncation(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: X\n---\n"+string(make([]byte, 5000))), 0600))

	s := parseSkill(skillDir)
	require.NotNil(t, s)

	content := loadSkillContent(*s, 100)
	require.Contains(t, content, "truncated")
}
