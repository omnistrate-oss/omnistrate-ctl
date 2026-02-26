package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	DefaultDir          string      = "~/.omnistrate"
	DefaultFile         string      = "config.yml"
	DefaultPermissions  os.FileMode = 0700
	GithubPATEnvVar     string      = "GH_PAT"   //nolint:gosec
	GithubTokenEnvVar   string      = "GH_TOKEN" //nolint:gosec
	GithubTokenUserName string      = "api"
)

// ConfigFile represents the Omnistrate CTL config file.
type ConfigFile struct {
	AuthConfigs               []AuthConfig `yaml:"auths"`
	GitHubPersonalAccessToken string       `yaml:"github_personal_access_token,omitempty"`
	FilePath                  string       `yaml:"-"`
}

// AuthConfig represents the authentication configuration.
type AuthConfig struct {
	Token string `yaml:"token,omitempty"`
}

var (
	ErrConfigFileNotFound = errors.New("auth failure: config file not found, please login first")
	ErrAuthConfigNotFound = errors.New("auth failure: auth credentials not found, please login first")
	ErrGitHubPATNotFound  = errors.New("no github personal access token found")
	ErrTokenExpired       = errors.New("auth failure: token has expired, please login again")
	ErrUnauthorized       = errors.New("auth failure: unauthorized access, please login again")
)

// New initializes a config file for the given file path.
func New(filePath string) (*ConfigFile, error) {
	if filePath == "" {
		return nil, fmt.Errorf("can't create config with empty filePath")
	}
	return &ConfigFile{
		AuthConfigs: make([]AuthConfig, 0),
		FilePath:    filePath,
	}, nil
}

// ConfigDir returns the path to the omnistrate-ctl config directory.
func ConfigDir() string {
	expandedDefaultDir, err := homedir.Expand(DefaultDir)
	if err != nil {
		return DefaultDir
	}
	return expandedDefaultDir
}

// EnsureFile creates the root directory and config file.
func EnsureFile() (string, error) {
	permission := DefaultPermissions
	dir := ConfigDir()
	dirPath, err := homedir.Expand(dir)
	if err != nil {
		return "", err
	}

	filePath := path.Clean(filepath.Join(dirPath, DefaultFile))
	if err = os.MkdirAll(filepath.Dir(filePath), permission); err != nil {
		return "", err
	}

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		var file *os.File
		file, err = os.OpenFile(filepath.Clean(filePath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return "", err
		}
		defer file.Close()
	}

	return filePath, nil
}

// fileExists checks if the config file exists.
func fileExists() bool {
	dir := ConfigDir()
	dirPath, err := homedir.Expand(dir)
	if err != nil {
		return false
	}

	filePath := path.Clean(filepath.Join(dirPath, DefaultFile))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// save writes the config to disk.
func (configFile *ConfigFile) save() error {
	file, err := os.OpenFile(configFile.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	var buff bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&buff)
	yamlEncoder.SetIndent(2)
	if err = yamlEncoder.Encode(configFile); err != nil {
		return err
	}

	_, err = file.Write(buff.Bytes())
	return err
}

// load reads the YAML file from disk.
func (configFile *ConfigFile) load() error {
	if _, err := os.Stat(configFile.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("can't load config from non existent filePath")
	}

	data, err := os.ReadFile(configFile.FilePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, configFile)
}

// CreateOrUpdateAuthConfig creates or updates the authentication configuration.
func CreateOrUpdateAuthConfig(authConfig AuthConfig) error {
	configPath, err := EnsureFile()
	if err != nil {
		return err
	}

	cfg, err := New(configPath)
	if err != nil {
		return err
	}

	if err = cfg.load(); err != nil {
		return err
	}

	if len(cfg.AuthConfigs) == 0 {
		cfg.AuthConfigs = append(cfg.AuthConfigs, authConfig)
	} else {
		cfg.AuthConfigs[0] = authConfig
	}

	return cfg.save()
}

// LookupAuthConfig returns the authentication configuration.
func LookupAuthConfig() (AuthConfig, error) {
	var authConfig AuthConfig

	if !fileExists() {
		return authConfig, ErrConfigFileNotFound
	}

	configPath, err := EnsureFile()
	if err != nil {
		return authConfig, err
	}

	cfg, err := New(configPath)
	if err != nil {
		return authConfig, err
	}

	if err = cfg.load(); err != nil {
		return authConfig, err
	}

	if len(cfg.AuthConfigs) > 0 && cfg.AuthConfigs[0].Token != "" {
		return cfg.AuthConfigs[0], nil
	}

	return authConfig, ErrAuthConfigNotFound
}

// RemoveAuthConfig deletes the authentication configuration.
func RemoveAuthConfig() error {
	if !fileExists() {
		return ErrConfigFileNotFound
	}

	configPath, err := EnsureFile()
	if err != nil {
		return err
	}

	cfg, err := New(configPath)
	if err != nil {
		return err
	}

	if err = cfg.load(); err != nil {
		return err
	}

	if len(cfg.AuthConfigs) > 0 {
		cfg.AuthConfigs = []AuthConfig{}
		return cfg.save()
	}

	return ErrAuthConfigNotFound
}

// CreateOrUpdateGitHubPersonalAccessToken creates or updates the authentication configuration.
func CreateOrUpdateGitHubPersonalAccessToken(gitHubPersonalAccessToken string) error {
	configPath, err := EnsureFile()
	if err != nil {
		return err
	}

	cfg, err := New(configPath)
	if err != nil {
		return err
	}

	if err = cfg.load(); err != nil {
		return err
	}

	cfg.GitHubPersonalAccessToken = gitHubPersonalAccessToken

	return cfg.save()
}

func IsGithubTokenEnvVarConfigured() bool {
	return os.Getenv(GithubTokenEnvVar) != ""
}

// LookupGitHubPersonalAccessToken returns the authentication configuration.
func LookupGitHubPersonalAccessToken() (string, error) {
	if os.Getenv(GithubTokenEnvVar) != "" {
		return os.Getenv(GithubTokenEnvVar), nil
	}
	if os.Getenv(GithubPATEnvVar) != "" {
		return os.Getenv(GithubPATEnvVar), nil
	}

	if !fileExists() {
		return "", ErrConfigFileNotFound
	}

	configPath, err := EnsureFile()
	if err != nil {
		return "", err
	}

	cfg, err := New(configPath)
	if err != nil {
		return "", err
	}

	if err = cfg.load(); err != nil {
		return "", err
	}

	if len(cfg.GitHubPersonalAccessToken) > 0 {
		return cfg.GitHubPersonalAccessToken, nil
	}

	return "", ErrGitHubPATNotFound
}

// RemoveGitHubPersonalAccessToken deletes the authentication configuration.
func RemoveGitHubPersonalAccessToken() error {
	if !fileExists() {
		return ErrConfigFileNotFound
	}

	configPath, err := EnsureFile()
	if err != nil {
		return err
	}

	cfg, err := New(configPath)
	if err != nil {
		return err
	}

	if err = cfg.load(); err != nil {
		return err
	}

	if len(cfg.GitHubPersonalAccessToken) > 0 {
		cfg.GitHubPersonalAccessToken = ""
		return cfg.save()
	}

	return ErrGitHubPATNotFound
}
