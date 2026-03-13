package agent

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentCommands(t *testing.T) {
	require := require.New(t)

	require.Equal("agent [operation] [flags]", Cmd.Use)
	require.Equal("Manage AI agent configurations and skills", Cmd.Short)

	expectedCommands := []string{"init", "chat"}
	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}
	for _, expected := range expectedCommands {
		require.Contains(actualCommands, expected)
	}
}

func TestChatCommandStructure(t *testing.T) {
	require := require.New(t)

	require.Equal("chat [provider]", chatCmd.Use)
	require.Contains(chatCmd.Short, "interactive AI agent chat")
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt("/tmp/test", "", "", nil)
	require.Contains(t, prompt, "/tmp/test")
	require.Contains(t, prompt, "omnistrate-ctl")
	require.Contains(t, prompt, "No Spec File Found")
}

func TestBuildSystemPromptWithSpec(t *testing.T) {
	specContent := "services:\n  redis:\n    image: redis:7\n"
	prompt := buildSystemPrompt("/tmp/test", "omnistrate-compose.yaml", specContent, nil)
	require.Contains(t, prompt, "omnistrate-compose.yaml")
	require.Contains(t, prompt, "redis:7")
	require.Contains(t, prompt, "Current Spec File")
}

func TestBuildSystemPromptWithSkill(t *testing.T) {
	s := &skill{
		Name:        "Test Skill",
		Description: "A test skill for unit testing",
	}
	prompt := buildSystemPrompt("/tmp/test", "", "", s)
	require.Contains(t, prompt, "Active Skill: Test Skill")
	require.Contains(t, prompt, "A test skill for unit testing")
}

func TestDetectSpec(t *testing.T) {
	tmpDir := t.TempDir()

	// No spec file
	name, content := detectSpec(tmpDir)
	require.Empty(t, name)
	require.Empty(t, content)

	// Create omnistrate-compose.yaml
	specContent := "services:\n  web:\n    image: nginx\n"
	err := os.WriteFile(tmpDir+"/omnistrate-compose.yaml", []byte(specContent), 0600)
	require.NoError(t, err)

	name, content = detectSpec(tmpDir)
	require.Equal(t, "omnistrate-compose.yaml", name)
	require.Equal(t, specContent, content)
}
