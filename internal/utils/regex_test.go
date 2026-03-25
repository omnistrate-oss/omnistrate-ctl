package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBuildContext(t *testing.T) {
	cwd, _ := os.Getwd()

	// Test cases with different scenarios.
	tests := []struct {
		name                   string
		input                  string
		versionTaggedImageURIs map[string]string
		expectedOutput         string
	}{
		{
			name: "Single build section",
			input: `
xxxxxx
    build:
      context: ./frontend
      dockerfile: Dockerfile.frontend
xxxx
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
			},
			expectedOutput: `
xxxxxx
    image: "ghcr.io/user/ai-chatbot-frontend:v1.0"
xxxx
`,
		},
		{
			name: "Multiple build sections",
			input: `
xxxxxx
    build:
      context: ./frontend
      dockerfile: Dockerfile.frontend
xxxx

another section

xxxxxx
    build:
      context: ./backend
      dockerfile: Dockerfile.backend
xxxx
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
				filepath.Join(cwd, "./backend", "Dockerfile.backend"):   "ghcr.io/user/ai-chatbot-backend:v1.0",
			},
			expectedOutput: `
xxxxxx
    image: "ghcr.io/user/ai-chatbot-frontend:v1.0"
xxxx

another section

xxxxxx
    image: "ghcr.io/user/ai-chatbot-backend:v1.0"
xxxx
`,
		},
		{
			name: "Missing entry in versionTaggedImageURIs",
			input: `
xxxxxx
    build:
      context: ./frontend
      dockerfile: Dockerfile.frontend
xxxx
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./backend", "Dockerfile.backend"): "ghcr.io/user/ai-chatbot-backend:v1.0", // Missing entry for frontend
			},
			expectedOutput: `
xxxxxx
    image: ""
xxxx
`,
		},
		{
			name:  "Empty input",
			input: ``,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
			},
			expectedOutput: ``,
		},
		{
			name: "No build section in input",
			input: `
some random content here without build section
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
			},
			expectedOutput: `
some random content here without build section
`,
		},
	}

	// Iterate over each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := ReplaceBuildContext(tt.input, tt.versionTaggedImageURIs)
			if actualOutput != tt.expectedOutput {
				t.Errorf("expected: %v, but got: %v", tt.expectedOutput, actualOutput)
			}
		})
	}
}
