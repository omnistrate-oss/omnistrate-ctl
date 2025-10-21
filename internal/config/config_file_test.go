package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	_, err := New("")
	assert.Error(t, err)

	cfg, err := New("test.yml")
	assert.NoError(t, err)
	assert.Equal(t, "test.yml", cfg.FilePath)
	assert.Empty(t, cfg.AuthConfigs)
}

func TestConfigDir(t *testing.T) {
	expandedDefaultDir, err := homedir.Expand(DefaultDir)
	assert.NoError(t, err)
	assert.Equal(t, expandedDefaultDir, ConfigDir())
}

func TestEnsureFile(t *testing.T) {
	filePath, err := EnsureFile()
	assert.NoError(t, err)
	assert.FileExists(t, filePath)

	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestFileExists(t *testing.T) {
	filePath, err := EnsureFile()
	assert.NoError(t, err)

	assert.True(t, fileExists())

	err = os.Remove(filePath)
	assert.NoError(t, err)
	assert.False(t, fileExists())
}

func TestSaveAndLoad(t *testing.T) {
	dir := ConfigDir()

	cfg, err := New(filepath.Join(dir, DefaultFile))
	assert.NoError(t, err)

	cfg.AuthConfigs = append(cfg.AuthConfigs, AuthConfig{
		Token: "token123",
	})

	err = cfg.save()
	assert.NoError(t, err)

	loadedCfg, err := New(filepath.Join(dir, DefaultFile))
	assert.NoError(t, err)

	err = loadedCfg.load()
	assert.NoError(t, err)

	assert.Equal(t, cfg.AuthConfigs, loadedCfg.AuthConfigs)

	err = RemoveAuthConfig()
	assert.NoError(t, err)
}

func TestAuthConfig(t *testing.T) {
	authConfig := AuthConfig{
		Token: "token123",
	}

	err := CreateOrUpdateAuthConfig(authConfig)
	assert.NoError(t, err)

	loadedConfig, err := LookupAuthConfig()
	assert.NoError(t, err)
	assert.Equal(t, authConfig, loadedConfig)

	err = RemoveAuthConfig()
	assert.NoError(t, err)

	_, err = LookupAuthConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrAuthConfigNotFound.Error())
}

func TestGitHubPersonalAccessToken(t *testing.T) {
	_, err := LookupGitHubPersonalAccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrGitHubPATNotFound.Error())

	err = CreateOrUpdateGitHubPersonalAccessToken("token123")
	assert.NoError(t, err)

	loadedPAT, err := LookupGitHubPersonalAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, "token123", loadedPAT)

	err = RemoveGitHubPersonalAccessToken()
	assert.NoError(t, err)

	_, err = LookupGitHubPersonalAccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrGitHubPATNotFound.Error())
}

func TestGitHubTokenFromEnvVar(t *testing.T) {
	t.Setenv("GH_TOKEN", "env_token")
	token, err := LookupGitHubPersonalAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, "env_token", token)
}

func TestGitHubPersonalAccessTokenFromEnvVar(t *testing.T) {
	t.Setenv("GH_PAT", "PAT_env_token")
	token, err := LookupGitHubPersonalAccessToken()
	assert.NoError(t, err)
	assert.Equal(t, "PAT_env_token", token)
}

func TestIsGithubTokenConfigured(t *testing.T) {
	isConfigured := IsGithubTokenEnvVarConfigured()
	assert.False(t, isConfigured)

	t.Setenv("GH_PAT", "PAT_env_token")
	isConfigured = IsGithubTokenEnvVarConfigured()
	assert.False(t, isConfigured)

	t.Setenv("GH_TOKEN", "env_token")
	isConfigured = IsGithubTokenEnvVarConfigured()
	assert.True(t, isConfigured)

	t.Setenv("GH_TOKEN", "")
	isConfigured = IsGithubTokenEnvVarConfigured()
	assert.False(t, isConfigured)
}

func TestLoadNonExistentFile(t *testing.T) {
	dir := ConfigDir()
	cfg, err := New(filepath.Join(dir, "non_existent.yml"))
	assert.NoError(t, err)

	err = cfg.load()
	assert.Error(t, err)
}

func TestLoadInvalidYaml(t *testing.T) {
	dir := ConfigDir()
	filePath := filepath.Join(dir, DefaultFile)
	err := os.WriteFile(filePath, []byte("invalid_yaml: [abc,"), 0600)
	assert.NoError(t, err)

	cfg, err := New(filePath)
	assert.NoError(t, err)

	err = cfg.load()
	assert.Error(t, err)

	err = os.Remove(filePath)
	assert.NoError(t, err)
}

func TestRemoveAuthConfigWhenFileNotExists(t *testing.T) {
	// Ensure config file doesn't exist
	dir := ConfigDir()
	filePath := filepath.Join(dir, DefaultFile)
	os.Remove(filePath)

	err := RemoveAuthConfig()
	assert.Error(t, err)
	assert.Equal(t, ErrConfigFileNotFound, err)
}

func TestRemoveAuthConfigWhenAuthNotExists(t *testing.T) {
	// Create empty config file
	dir := ConfigDir()
	filePath := filepath.Join(dir, DefaultFile)
	err := os.WriteFile(filePath, []byte("auths: []\n"), 0600)
	assert.NoError(t, err)

	err = RemoveAuthConfig()
	assert.Error(t, err)
	assert.Equal(t, ErrAuthConfigNotFound, err)

	// Cleanup
	os.Remove(filePath)
}
