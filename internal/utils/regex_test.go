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
		{
			name: "Build section with cache_from and cache_to",
			input: `
services:
  hello-world:
    build:
      context: ./frontend
      dockerfile: Dockerfile.frontend
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
    environment:
      FOO: bar
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
			},
			expectedOutput: `
services:
  hello-world:
    image: "ghcr.io/user/ai-chatbot-frontend:v1.0"
    environment:
      FOO: bar
`,
		},
		{
			name: "Build section with cache_from only",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
      cache_from:
        - type=gha
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v2.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v2.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Cache fields before context and dockerfile",
			input: `
services:
  hello-world:
    build:
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
      context: ./frontend
      dockerfile: Dockerfile.frontend
    environment:
      FOO: bar
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile.frontend"): "ghcr.io/user/ai-chatbot-frontend:v1.0",
			},
			expectedOutput: `
services:
  hello-world:
    image: "ghcr.io/user/ai-chatbot-frontend:v1.0"
    environment:
      FOO: bar
`,
		},
		{
			name: "Cache between context and dockerfile",
			input: `
services:
  api:
    build:
      context: ./api
      cache_from:
        - type=gha
        - type=registry,ref=ghcr.io/owner/api:cache
      dockerfile: Dockerfile
      cache_to:
        - type=gha,mode=max
    ports:
      - "3000:3000"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./api", "Dockerfile"): "ghcr.io/user/api:v1.0",
			},
			expectedOutput: `
services:
  api:
    image: "ghcr.io/user/api:v1.0"
    ports:
      - "3000:3000"
`,
		},
		{
			name: "Multiple services with mixed cache positions",
			input: `
services:
  web:
    build:
      cache_from:
        - type=gha
      context: ./frontend
      dockerfile: Dockerfile
      cache_to:
        - type=gha,mode=max
    ports:
      - "8080:8080"
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile.api
      cache_from:
        - type=gha
    environment:
      DB_HOST: localhost
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile"):    "ghcr.io/user/web:v1.0",
				filepath.Join(cwd, "./backend", "Dockerfile.api"): "ghcr.io/user/api:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
  api:
    image: "ghcr.io/user/api:v1.0"
    environment:
      DB_HOST: localhost
`,
		},
		// --- Additional regression tests ---
		{
			name: "Build with args field preserved (no context/dockerfile = keep as-is)",
			input: `
services:
  web:
    build:
      args:
        NODE_ENV: production
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{},
			expectedOutput: `
services:
  web:
    build:
      args:
        NODE_ENV: production
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build with context only (no dockerfile) is kept",
			input: `
services:
  web:
    build:
      context: ./app
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{},
			expectedOutput: `
services:
  web:
    build:
      context: ./app
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build with dockerfile only (no context) is kept",
			input: `
services:
  web:
    build:
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{},
			expectedOutput: `
services:
  web:
    build:
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build with args alongside context and dockerfile",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
      args:
        NODE_ENV: production
        VERSION: "1.0"
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build with target and shm_size",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
      target: builder
      shm_size: '2gb'
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build keyword in a string value is not replaced",
			input: `
services:
  web:
    image: "myapp:latest"
    environment:
      BUILD_MODE: "build: context"
      DESCRIPTION: "This will build: things"
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{},
			expectedOutput: `
services:
  web:
    image: "myapp:latest"
    environment:
      BUILD_MODE: "build: context"
      DESCRIPTION: "This will build: things"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Blank line inside build block is consumed",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile

      cache_from:
        - type=gha
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Multiple blank lines inside build block",
			input: `
services:
  web:
    build:
      context: ./app


      dockerfile: Dockerfile
      cache_from:
        - type=gha
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build block at end of file without trailing newline",
			input: `services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
      cache_from:
        - type=gha`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `services:
  web:
    image: "ghcr.io/user/web:v1.0"`,
		},
		{
			name: "Build at different indentation levels (2 vs 4 spaces)",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
  api:
        build:
            context: ./api
            dockerfile: Dockerfile.api
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"):     "ghcr.io/user/web:v1.0",
				filepath.Join(cwd, "./api", "Dockerfile.api"): "ghcr.io/user/api:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
  api:
        image: "ghcr.io/user/api:v1.0"
`,
		},
		{
			name: "Real-world ray-cluster spec layout",
			input: `
version: '3'
services:
  hello-world:
    build:
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
      context: ./jobs/hello-world
      dockerfile: Dockerfile
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
    deploy:
      resources:
        reservations:
          cpus: "0.1"
          memory: 256M
    x-omnistrate-compute:
      replicaCountApiParam: numReplicas
  hello-cuda:
    build:
      context: ./jobs/hello-cuda
      dockerfile: Dockerfile
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
x-omnistrate-image-registry-attributes:
  ghcr.io:
    auth:
      password: secret
      username: user
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./jobs/hello-world", "Dockerfile"): "ghcr.io/org/ray-cluster-hello-world:sha-abc123",
				filepath.Join(cwd, "./jobs/hello-cuda", "Dockerfile"):  "ghcr.io/org/ray-cluster-hello-cuda:sha-def456",
			},
			expectedOutput: `
version: '3'
services:
  hello-world:
    image: "ghcr.io/org/ray-cluster-hello-world:sha-abc123"
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
    deploy:
      resources:
        reservations:
          cpus: "0.1"
          memory: 256M
    x-omnistrate-compute:
      replicaCountApiParam: numReplicas
  hello-cuda:
    image: "ghcr.io/org/ray-cluster-hello-cuda:sha-def456"
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
x-omnistrate-image-registry-attributes:
  ghcr.io:
    auth:
      password: secret
      username: user
`,
		},
		{
			name: "Service with image field is untouched",
			input: `
services:
  redis:
    image: redis:7
    ports:
      - "6379:6379"
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  redis:
    image: redis:7
    ports:
      - "6379:6379"
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Build with multiple cache_from entries",
			input: `
services:
  web:
    build:
      context: ./app
      dockerfile: Dockerfile
      cache_from:
        - type=gha
        - type=registry,ref=ghcr.io/user/web:cache
        - type=local,src=/tmp/.buildx-cache
      cache_to:
        - type=gha,mode=max
        - type=registry,ref=ghcr.io/user/web:cache
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Dockerfile reversed before context",
			input: `
services:
  web:
    build:
      dockerfile: Dockerfile
      context: ./app
    ports:
      - "8080:8080"
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./app", "Dockerfile"): "ghcr.io/user/web:v1.0",
			},
			expectedOutput: `
services:
  web:
    image: "ghcr.io/user/web:v1.0"
    ports:
      - "8080:8080"
`,
		},
		{
			name: "Three services one without build",
			input: `
services:
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      cache_from:
        - type=gha
    ports:
      - "3000:3000"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  backend:
    build:
      cache_to:
        - type=gha,mode=max
      context: ./backend
      dockerfile: Dockerfile.prod
      cache_from:
        - type=gha
    environment:
      REDIS_URL: redis://redis:6379
`,
			versionTaggedImageURIs: map[string]string{
				filepath.Join(cwd, "./frontend", "Dockerfile"):     "ghcr.io/user/frontend:v1.0",
				filepath.Join(cwd, "./backend", "Dockerfile.prod"): "ghcr.io/user/backend:v1.0",
			},
			expectedOutput: `
services:
  frontend:
    image: "ghcr.io/user/frontend:v1.0"
    ports:
      - "3000:3000"
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  backend:
    image: "ghcr.io/user/backend:v1.0"
    environment:
      REDIS_URL: redis://redis:6379
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
