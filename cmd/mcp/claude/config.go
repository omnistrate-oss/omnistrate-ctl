package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
)

// Config represents the structure of Claude's desktop configuration
type Config struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents an MCP server configuration entry
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Manager handles Claude MCP configuration file operations
type Manager struct {
	configPath string
}

// NewManager creates a new config manager with the default or specified path
func NewManager(configPath string) *Manager {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}
	return &Manager{
		configPath: configPath,
	}
}

// getDefaultConfigPath returns the default Claude config path based on OS
func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	default: // linux and others
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configHome, "Claude", "claude_desktop_config.json")
	}
}

// LoadConfig loads the Claude configuration from file
func (m *Manager) LoadConfig() (*Config, error) {
	config := &Config{
		MCPServers: make(map[string]MCPServer),
	}

	if err := cfgmgr.LoadJSONConfig(m.configPath, config); err != nil {
		return nil, fmt.Errorf("failed to load Claude configuration: %w", err)
	}

	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}

	return config, nil
}

// SaveConfig saves the Claude configuration to file
func (m *Manager) SaveConfig(config *Config) error {
	if err := cfgmgr.SaveJSONConfig(m.configPath, config); err != nil {
		return fmt.Errorf("failed to save Claude configuration: %w", err)
	}
	return nil
}

// AddServer adds or updates an MCP server configuration
func (m *Manager) AddServer(name string, server MCPServer) error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	config.MCPServers[name] = server
	return m.SaveConfig(config)
}

// RemoveServer removes an MCP server configuration
func (m *Manager) RemoveServer(name string) error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	delete(config.MCPServers, name)
	return m.SaveConfig(config)
}

// HasServer checks if a server with the given name exists
func (m *Manager) HasServer(name string) (bool, error) {
	config, err := m.LoadConfig()
	if err != nil {
		return false, err
	}

	_, exists := config.MCPServers[name]
	return exists, nil
}

// GetConfigPath returns the path to the Claude configuration file being used
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
	return config.MCPServers, nil
}
