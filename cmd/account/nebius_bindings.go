package account

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"gopkg.in/yaml.v3"
)

type nebiusBindingsFile struct {
	Bindings       []nebiusBindingFileEntry `yaml:"bindings"`
	NebiusBindings []nebiusBindingFileEntry `yaml:"nebiusBindings"`
}

type nebiusBindingFileEntry struct {
	ProjectID          string `yaml:"projectID"`
	ProjectId          string `yaml:"projectId"`
	PublicKeyID        string `yaml:"publicKeyID"`
	PublicKeyId        string `yaml:"publicKeyId"`
	ServiceAccountID   string `yaml:"serviceAccountID"`
	ServiceAccountId   string `yaml:"serviceAccountId"`
	PrivateKeyPEM      string `yaml:"privateKeyPEM"`
	PrivateKeyPem      string `yaml:"privateKeyPem"`
	PrivateKeyPEMFile  string `yaml:"privateKeyPEMFile"`
	PrivateKeyPemFile  string `yaml:"privateKeyPemFile"`
	PrivateKeyFile     string `yaml:"privateKeyFile"`
	PrivateKeyFilePath string `yaml:"privateKeyFilePath"`
}

func parseNebiusBindingsFile(path string) ([]openapiclient.NebiusAccountBindingInput, error) {
	resolvedPath, err := expandAndResolvePath(path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Nebius bindings file path %q: %w", path, err)
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Nebius bindings file %q: %w", resolvedPath, err)
	}

	var wrapped nebiusBindingsFile
	if err := yaml.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse Nebius bindings file %q: %w", resolvedPath, err)
	}

	entries := wrapped.Bindings
	if len(entries) == 0 {
		entries = wrapped.NebiusBindings
	}
	if len(entries) == 0 {
		var directEntries []nebiusBindingFileEntry
		if err := yaml.Unmarshal(data, &directEntries); err == nil && len(directEntries) > 0 {
			entries = directEntries
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("Nebius bindings file %q must contain at least one binding", resolvedPath)
	}

	baseDir := filepath.Dir(resolvedPath)
	bindings := make([]openapiclient.NebiusAccountBindingInput, 0, len(entries))
	seenProjectIDs := make(map[string]struct{}, len(entries))

	for index, entry := range entries {
		binding, err := entry.toAPI(baseDir)
		if err != nil {
			return nil, fmt.Errorf("invalid Nebius binding at index %d: %w", index, err)
		}

		key := strings.ToLower(binding.ProjectID)
		if _, exists := seenProjectIDs[key]; exists {
			return nil, fmt.Errorf("duplicate Nebius binding for project %q", binding.ProjectID)
		}
		seenProjectIDs[key] = struct{}{}

		bindings = append(bindings, binding)
	}

	return bindings, nil
}

func (entry nebiusBindingFileEntry) toAPI(baseDir string) (openapiclient.NebiusAccountBindingInput, error) {
	projectID := firstNonEmpty(entry.ProjectID, entry.ProjectId)
	publicKeyID := firstNonEmpty(entry.PublicKeyID, entry.PublicKeyId)
	serviceAccountID := firstNonEmpty(entry.ServiceAccountID, entry.ServiceAccountId)
	privateKeyPEM := strings.TrimSpace(firstNonEmpty(entry.PrivateKeyPEM, entry.PrivateKeyPem))
	privateKeyPEMFile := firstNonEmpty(
		entry.PrivateKeyPEMFile,
		entry.PrivateKeyPemFile,
		entry.PrivateKeyFile,
		entry.PrivateKeyFilePath,
	)

	if privateKeyPEM != "" && privateKeyPEMFile != "" {
		return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("specify only one of privateKeyPEM or privateKeyPEMFile")
	}

	if privateKeyPEM == "" {
		if privateKeyPEMFile == "" {
			return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("privateKeyPEMFile is required")
		}

		resolvedKeyPath, err := expandAndResolvePath(privateKeyPEMFile, baseDir)
		if err != nil {
			return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("failed to resolve private key path %q: %w", privateKeyPEMFile, err)
		}

		keyData, err := os.ReadFile(resolvedKeyPath)
		if err != nil {
			return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("failed to read private key file %q: %w", resolvedKeyPath, err)
		}

		privateKeyPEM = strings.TrimSpace(string(keyData))
		if privateKeyPEM == "" {
			return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("private key file %q is empty", resolvedKeyPath)
		}
	}

	switch {
	case strings.TrimSpace(projectID) == "":
		return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("projectID is required")
	case strings.TrimSpace(publicKeyID) == "":
		return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("publicKeyID is required")
	case strings.TrimSpace(serviceAccountID) == "":
		return openapiclient.NebiusAccountBindingInput{}, fmt.Errorf("serviceAccountID is required")
	}

	return openapiclient.NebiusAccountBindingInput{
		PrivateKeyPEM:    privateKeyPEM,
		ProjectID:        strings.TrimSpace(projectID),
		PublicKeyID:      strings.TrimSpace(publicKeyID),
		ServiceAccountID: strings.TrimSpace(serviceAccountID),
	}, nil
}

func expandAndResolvePath(path string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	expanded, err := homedir.Expand(trimmed)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(expanded) {
		if baseDir == "" {
			baseDir = "."
		}
		expanded = filepath.Join(baseDir, expanded)
	}

	return filepath.Clean(expanded), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
