# CLAUDE.md - CLI Development Guidelines

## Build Commands

- Build: `make build`
- Run all tests: `make unit-test`
- Run single test: `go test ./path/to/package -run TestName`
- Run smoke tests: `make smoke-test` (requires TEST_EMAIL, TEST_PASSWORD)
- Lint code: `make lint` (install with `make lint-install`)
- Format code: `make pretty` (uses prettier and go fmt)
- Generate docs: `make gen-doc` (regenerates CLI documentation)
- Run everything: `make all` (includes tidy, build, test, lint, check-dependencies, gen-doc, pretty)

## Documentation Generation Requirements

**IMPORTANT**: After making ANY changes to CLI commands, flags, or help text, you MUST run `make gen-doc` to regenerate the documentation. This ensures the docs in `mkdocs/docs/` stay synchronized with the actual CLI behavior.

- The `gen-doc` target runs `go run doc-gen/main.go` which auto-generates markdown files
- Documentation files are automatically removed and regenerated to stay current
- Always run `make gen-doc` or `make all` after modifying:
  - Command definitions in `cmd/` directory
  - Flag descriptions or help text
  - Command usage examples
  - Any cobra command configurations

## Code Style Guidelines

- Follow Go standard naming conventions: camelCase for variables, PascalCase for exports
- Use error handling with appropriate checks - wrap errors with context when needed
- Format imports with standard Go style (stdlib first, then external, then internal)
- Use descriptive variable/function names and maintain consistent indentation
- Add tests for all new functionality and maintain high coverage
- Follow project structure with cmd/ for commands and internal/ for implementation
- Commit messages should be clear and descriptive (feature/fix/chore: message)

## Project-Specific Patterns

- Prefer functional options for configuration
- Use cobra for CLI commands with consistent flags/args pattern
- Use tabwriter for formatted terminal output
