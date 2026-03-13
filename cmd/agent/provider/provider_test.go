package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderRegistry(t *testing.T) {
	// All three providers should be registered via init()
	names := Names()
	require.Contains(t, names, "claude")
	require.Contains(t, names, "chatgpt")
	require.Contains(t, names, "copilot")
}

func TestGetProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{"claude exists", "claude", false},
		{"chatgpt exists", "chatgpt", false},
		{"copilot exists", "copilot", false},
		{"unknown provider", "gemini", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Get(tt.provider)
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, p)
			} else {
				require.NoError(t, err)
				require.NotNil(t, p)
			}
		})
	}
}

func TestProviderNames(t *testing.T) {
	tests := []struct {
		providerKey  string
		expectedName string
	}{
		{"claude", "Claude"},
		{"chatgpt", "ChatGPT"},
		{"copilot", "Copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.providerKey, func(t *testing.T) {
			p, err := Get(tt.providerKey)
			require.NoError(t, err)
			require.Equal(t, tt.expectedName, p.Name())
		})
	}
}

func TestProviderIsConfigured(t *testing.T) {
	// Without env vars set, providers should not be configured
	tests := []struct {
		providerKey string
	}{
		{"claude"},
		{"chatgpt"},
		{"copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.providerKey, func(t *testing.T) {
			p, err := Get(tt.providerKey)
			require.NoError(t, err)

			ok, hint := p.IsConfigured()
			// We can't guarantee env vars are set in CI, but we can check the interface works
			if !ok {
				require.NotEmpty(t, hint, "hint should explain what's missing")
			}
		})
	}
}
