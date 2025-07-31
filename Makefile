GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
TAG?=latest

GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_VERSION=$(shell git describe --tags 2>/dev/null || echo "$(GIT_COMMIT)")
GIT_UNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no | grep -v 'go.sum')
ifneq ($(GIT_UNTRACKEDCHANGES),)
	GIT_VERSION := $(GIT_VERSION)-dirty
	GIT_COMMIT := $(GIT_COMMIT)-dirty
endif

GIT_USER?=$(shell gh api user -q ".login") # gets current user using github cli if the variable is not already set
GIT_TOKEN?=$(shell gh config get -h github.com oauth_token) # gets current user using github cli if the variable is not already set
PROJECT_NAME=omnistrate-ctl
DOCKER_PLATFORM=linux/arm64 
TESTCOVERAGE_THRESHOLD=0
REPO_ROOT=$(shell git rev-parse --show-toplevel)

# Build info
BUILD_INFO_PKG=github.com/omnistrate-oss/omnistrate-ctl/internal/config
BUILD_TIMESTAMP=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_FLAGS=-trimpath -ldflags "-X $(BUILD_INFO_PKG).CommitID=$(GIT_COMMIT) -X $(BUILD_INFO_PKG).Timestamp=$(BUILD_TIMESTAMP) -X $(BUILD_INFO_PKG).Version=$(GIT_VERSION)"

CGO_ENABLED=0
GOPRIVATE=github.com/omnistrate

.PHONY: all
all: tidy build unit-test lint check-dependencies gen-doc pretty

.PHONE: pretty 
pretty:
	@echo "Running go fmt"
	@npx prettier --write .
	
.PHONY: tidy
tidy:
	@echo "Tidy dependency modules"
	go mod tidy

.PHONY: download
download:
	@echo "Download dependency modules"
	go mod download

.PHONY: unit-test
unit-test:
	@echo "Running unit tests for service"
	go test ./... -skip ./test/... $(ARGS) -cover -coverprofile coverage.out -covermode count
	go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/[%]//g' | awk 'current=$$1; {if (current < ${TESTCOVERAGE_THRESHOLD}) {print "\033[31mTest coverage is " current " which is below threshold\033[0m"; exit 1} else {print "\033[32mTest coverage is above threshold\033[0m"}}'

.PHONY: smoke-test
smoke-test:
	@echo "Running smoke tests for service"
	@echo you need to set the following environment variables: TEST_EMAIL, TEST_PASSWORD before running the smoke tests
	export ENABLE_SMOKE_TEST=true && \
	export OMNISTRATE_ROOT_DOMAIN=omnistrate.dev && \
	export OMNISTRATE_LOG_LEVEL=debug && \
	export OMNISTRATE_LOG_FORMAT=pretty && \
	go clean -testcache && \
	go test ./... -skip ./test/smoke_test/... $(ARGS) 

.PHONY: integration-test
integration-test:
	@echo "Running integration tests for service"
	@echo you need to set the following environment variables: TEST_EMAIL, TEST_PASSWORD before running the integration tests
	export ENABLE_INTEGRATION_TEST=true && \
	export OMNISTRATE_ROOT_DOMAIN=omnistrate.dev && \
	export OMNISTRATE_LOG_LEVEL=debug && \
	export OMNISTRATE_LOG_FORMAT=pretty && \
	go clean -testcache && \
	go test ./... -skip ./test/integration_test/... $(ARGS) 

.PHONY: build
build:
	@echo "Building CTL for $(GOOS)-$(GOARCH)"
	@binary_name="omnistrate-ctl-$(GOOS)-$(GOARCH)"; \
	if [ "$(GOOS)" = "windows" ]; then \
		binary_name="$$binary_name.exe"; \
	fi; \
	CGO_ENABLED=0 go build -mod=mod ${BUILD_FLAGS} -o dist/$$binary_name github.com/omnistrate-oss/omnistrate-ctl
	@echo "Build complete: dist/$$binary_name"
	@echo "Build integration test"
	go test -c -o /dev/null ./test/integration_test/...
	
.PHONY: ctl-linux-amd64
ctl-linux-amd64: main.go
	GOOS=linux GOARCH=amd64 make build

.PHONY: ctl-linux-arm64
ctl-linux-arm64: main.go
	GOOS=linux GOARCH=arm64 make build

.PHONY: ctl-darwin-amd64
ctl-darwin-amd64: main.go
	GOOS=darwin GOARCH=amd64 make build

.PHONY: ctl-darwin-arm64
ctl-darwin-arm64: main.go
	GOOS=darwin GOARCH=arm64 make build

.PHONY: ctl-windows-amd64
ctl-windows-amd64: main.go
	GOOS=windows GOARCH=amd64 make build

.PHONY: ctl-windows-arm64
ctl-windows-arm64: main.go
	GOOS=windows GOARCH=arm64 make build

.PHONY: ctl
ctl: ctl-linux-amd64 ctl-linux-arm64 ctl-darwin-amd64 ctl-darwin-arm64 ctl-windows-amd64 ctl-windows-arm64

.PHONY: test-coverage-report
test-coverage-report:
	go test ./... -skip ./test/... -cover -coverprofile coverage.out -covermode count
	go tool cover -html=coverage.out

lint-install:
	@echo "Installing golangci-lint"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2

.PHONY: lint
lint:
	@echo "Running checks for service"
	golangci-lint run ./...

.PHONY: sec-install
sec-install:
	@echo "Installing gosec"
	go install github.com/securego/gosec/v2/cmd/gosec@v2.18.0
.PHONY: sec
sec:
	@echo "Security scanning for service"
	gosec -tests --quiet ./...
.PHONY: sec-verbose
sec-verbose:
	@echo "Security scanning for service"
	gosec -tests ./...

.PHONY: update-dependencies
update-dependencies:
	@echo "Updating dependencies"
	go get -t -u ./...
	make tidy

.PHONY: update-omnistrate-dependencies
update-omnistrate-dependencies:
	@echo "Updating omnistrate dependencies"
	go get -u github.com/omnistrate/...
	go get -u github.com/omnistrate-oss/...
	make tidy

.PHONY: check-dependencies
check-dependencies:
	@echo "Checking dependencies starting with github.com/omnistrate..."
	@violating_deps=$$(grep -E '^\s*github.com/omnistrate/' go.mod | grep -v -E 'github.com/omnistrate-oss/'); \
    if [ -n "$$violating_deps" ]; then \
        echo "Error: Found dependencies starting with github.com/omnistrate/commons other than allowed ones:"; \
        echo "$$violating_deps"; \
        exit 1; \
    else \
        echo "No conflicting dependencies found."; \
    fi

.PHONY: gen-doc
gen-doc:
	@echo "Generating docs"
	rm -f mkdocs/docs/omnistrate-ctl*.md # remove old docs
	go run doc-gen/main.go

.PHONY: docker
docker: docker-build
.PHONY: docker-build
docker-build:
	docker build --platform=${DOCKER_PLATFORM} --build-arg GIT_USER=${GIT_USER} --build-arg GIT_TOKEN=${GIT_TOKEN} --build-arg GIT_COMMIT=${GIT_COMMIT} --build-arg GIT_VERSION=${GIT_VERSION} -f ./build/Dockerfile  -t ${PROJECT_NAME}:latest .

.PHONY: docker-run
docker-run:
	docker run --platform=${DOCKER_PLATFORM} -t ${PROJECT_NAME}:latest

# Other
.PHONY: clean
clean:
	@echo "Cleaning up"
	rm ./omnistrate-ctl
	rm ./coverage.out
	rm ./coverage-report.html
	rm ./coverage-report.txt
	rm ./test-report.json
	rm ./security-report.html
	rm ./docs

.PHONY: docker-build-docs
docker-build-docs:
	@echo "Building mkdocs-ctl-manual docker image"
	@docker build -t mkdocs-ctl-manual.local -f ./build/Dockerfile.docs.local .

.PHONY: docker-run-docs
docker-run-docs:
	@make docker-build-docs
	@echo "Starting mkdocs-ctl-manual.local on port 8001"
	@docker run -it --rm -p 8001:8001 mkdocs-ctl-manual.local

.PHONY: docker-run-rendered
docker-run-rendered:
	@echo "Starting mkdocs on port 8001 in rendered mode"
	@docker build -f ./build/Dockerfile.docs -t mkdocs-ctl-manual .
	@docker run -it --rm -p 8001:8001 mkdocs-ctl-manual
