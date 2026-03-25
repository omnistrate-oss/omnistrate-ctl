package vscode

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
)

// Config represents the structure of VSCode's MCP configuration
type Config struct {
	Inputs  []Input              `json:"inputs,omitempty"`
	Servers map[string]MCPServer `json:"servers"`
}

// Input represents an input variable configuration
type Input struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Description string `json:"description"`
	Password    bool   `json:"password,omitempty"`
}

// MCPServer represents an MCP server configuration entry for VSCode
type MCPServer struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ConfigType represents the type of VSCode configuration
type ConfigType int

const (
	// WorkspaceConfig represents workspace-specific configuration (.vscode/mcp.json)
	WorkspaceConfig ConfigType = iota
	// UserConfig represents user-global configuration (mcp.json)
	UserConfig
)

// Manager handles VSCode MCP configuration file operations
type Manager struct {
	configPath string
	configType ConfigType
}

// NewManager creates a new config manager with the default or specified path
func NewManager(configPath string, configType ConfigType) *Manager {
	if configPath == "" {
		switch configType {
		case WorkspaceConfig:
			configPath = getDefaultWorkspaceConfigPath()
		case UserConfig:
			configPath = getDefaultUserConfigPath()
		}
	}
	return &Manager{
		configPath: configPath,
		configType: configType,
	}
}

// getDefaultWorkspaceConfigPath returns the default workspace config path (.vscode/mcp.json)
func getDefaultWorkspaceConfigPath() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".vscode", "mcp.json")
	}
	return filepath.Join(workingDir, ".vscode", "mcp.json")
}

// getDefaultUserConfigPath returns the default user config path based on OS
func getDefaultUserConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "mcp.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Code", "User", "mcp.json")
	default: // linux
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configHome, "Code", "User", "mcp.json")
	}
}

// LoadConfig loads the VSCode configuration from file
func (m *Manager) LoadConfig() (*Config, error) {
	config := &Config{
		Servers: make(map[string]MCPServer),
	}

	if err := cfgmgr.LoadJSONConfig(m.configPath, config); err != nil {
		return nil, fmt.Errorf("failed to load VSCode configuration: %w", err)
	}

	if config.Servers == nil {
		config.Servers = make(map[string]MCPServer)
	}

	return config, nil
}

// SaveConfig saves the VSCode configuration to file
func (m *Manager) SaveConfig(config *Config) error {
	if err := cfgmgr.SaveJSONConfig(m.configPath, config); err != nil {
		return fmt.Errorf("failed to save VSCode configuration: %w", err)
	}
	return nil
}

// AddServer adds or updates an MCP server configuration
func (m *Manager) AddServer(name string, server MCPServer) error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	config.Servers[name] = server
	return m.SaveConfig(config)
}

// RemoveServer removes an MCP server configuration
func (m *Manager) RemoveServer(name string) error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	delete(config.Servers, name)
	return m.SaveConfig(config)
}

// HasServer checks if a server with the given name exists
func (m *Manager) HasServer(name string) (bool, error) {
	config, err := m.LoadConfig()
	if err != nil {
		return false, err
	}

	_, exists := config.Servers[name]
	return exists, nil
}

// GetConfigPath returns the path to the VSCode configuration file being used
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// BackupConfig creates a backup of the current configuration file
func (m *Manager) BackupConfig() error {
	return cfgmgr.BackupConfigFile(m.configPath)
}

// ListServers returns all configured MCP servers
func (m *Manager) ListServers() (map[string]MCPServer, error) {
	config, err := m.LoadConfig()
	if err != nil {
		return nil, err
	}
	return config.Servers, nil
}
