package cfgmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
)

const (
	MCPCommandName   = "mcp"
	StartCommandName = "start"
)

// ValidateExecutable validates that the given path points to an executable file.
// Unlike the ophis library version, this function preserves the original path
// (including symlinks) to ensure configuration remains valid after upgrades.
func ValidateExecutable(executablePath string) (string, error) {
	// Resolve symlinks only for validation purposes
	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable symlinks at '%s': %w", executablePath, err)
	}

	// Validate using the resolved path
	stat, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("executable not found at path '%s': ensure the binary is built and accessible", resolvedPath)
		}
		return "", fmt.Errorf("failed to access executable at '%s': %w", resolvedPath, err)
	}

	if stat.Mode()&0o111 == 0 {
		return "", fmt.Errorf("file at '%s' is not executable: check file permissions", resolvedPath)
	}

	// Return the ORIGINAL path to preserve symlinks
	// This ensures configuration remains valid after package upgrades (e.g., Homebrew)
	return executablePath, nil
}

// DeriveServerName derives a server name from an executable path.
func DeriveServerName(executablePath string) string {
	serverName := filepath.Base(executablePath)
	if ext := filepath.Ext(serverName); ext != "" {
		serverName = serverName[:len(serverName)-len(ext)]
	}
	return serverName
}

// GetMCPCommandPath constructs the command path for invoking the MCP server.
func GetMCPCommandPath(cmd *cobra.Command) []string {
	args := []string{}
	foundMCP := false
	cur := cmd
	for {
		if cur.Name() == MCPCommandName {
			foundMCP = true
		}
		if foundMCP {
			args = append(args, cur.Name())
		}
		if cur.Parent() == nil {
			break
		}
		cur = cur.Parent()
	}

	if len(args) == 0 {
		return []string{}
	}

	slices.Reverse(args)

	return args[1:]
}

// BackupConfigFile creates a backup of a configuration file.
func BackupConfigFile(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	backupPath := configPath + ".backup"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file for backup at '%s': %w", configPath, err)
	}

	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write backup configuration file at '%s': %w", backupPath, err)
	}

	return nil
}

// LoadJSONConfig loads a JSON configuration file into the provided interface.
func LoadJSONConfig(configPath string, config interface{}) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file at '%s': %w", configPath, err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse configuration file at '%s': invalid JSON format: %w", configPath, err)
	}

	return nil
}

// SaveJSONConfig saves a configuration to a JSON file with proper formatting.
func SaveJSONConfig(configPath string, config interface{}) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create configuration directory at '%s': %w", filepath.Dir(configPath), err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to JSON: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write configuration file at '%s': %w", configPath, err)
	}

	return nil
}
