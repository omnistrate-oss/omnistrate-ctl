# CLAUDE Development Context for ctl

*Last Updated: 2025-09-30T02:00:13.031428*

## 🎯 Service Purpose & Context
ctl is a key microservice in the Omnistrate platform ecosystem.

## 📊 Repository Metrics
- **Total Commits**: 355
- **Active Branch**: testchartvalues
- **Technologies**: Go, Make, GitHub Actions
- **Health Status**: healthy

## 🏗️ Architecture & Patterns

### Technology Stack
- **Go**: Primary development language with modern Go patterns
- **Make**: Build automation and development workflow
- **GitHub Actions**: Continuous integration and deployment

### File Organization
```
ctl/
├── cmd/                 # Entry points
├── pkg/                 # Main packages
├── config/              # Configuration
├── test/                # Tests
├── scripts/             # Build scripts
└── deploy/              # Deployment configs
```

## 🔄 Recent Development Activity

### Latest Commits (Last 30 Days)
**2025-09-26** `15402fd` Adding layered value file to smoke test
  *by maziarkaveh*

**2025-09-26** `3cc2149` Merge branch 'main' of github.com:omnistrate/ctl into testchartvalues
  *by maziarkaveh*

**2025-09-26** `03a38c2` Adding layered value file to smoke test (#458)
  *by maziar kaveh*

**2025-09-26** `3787267` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `f9213c3` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `a540a26` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `e2badc4` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `f527d33` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `5412353` helm_chart_layered_values_test.go
  *by maziarkaveh*

**2025-09-26** `63128c9` fix request (#456)
  *by Yuhui*

### Active Development Areas
- .dockerignore
- .github/workflows/build-docs.yaml
- .github/workflows/build.yml
- .github/workflows/docs-deploy.yml
- .github/workflows/package.yml
- .github/workflows/release.yml
- .github/workflows/smoke-tests.yml
- .gitignore
- AGENTS.md
- Makefile

## 👥 Development Team Context

### Contributors & Expertise
- **pberton**: 126 commits (High activity)
- **Xinyi**: 103 commits (High activity)
- **Alok**: 35 commits (Medium activity)
- **Yuhui**: 25 commits (Medium activity)
- **dependabot[bot]**: 16 commits (Medium activity)
- **maziar kaveh**: 15 commits (Medium activity)
- **Tomislav Simeunovic**: 10 commits (Light activity)
- **maziarkaveh**: 8 commits (Light activity)

## 🛠️ Development Workflow

### Build & Test Pipeline
```bash
# Standard workflow for ctl
make tidy              # Clean dependencies
make build             # Build service
make test              # Run tests
make lint              # Code quality
make integration-test  # Integration tests (if available)
```

### Code Quality Metrics
- **Test Coverage**: 49 test files
- **Code Size**: 366.5 MB
- **Go Files**: 5940 files

## 🔍 Code Analysis Insights

### Health Assessment
**Overall Health**: Healthy

✅ Service follows best practices with no detected issues.

### Recommended Focus Areas
- Increase test coverage (current ratio suggests room for improvement)
- High activity detected - ensure code review quality

## 🚀 AI Assistant Guidelines

When developing for ctl:

1. **Context Awareness**: Review recent commits to understand current development direction
2. **Pattern Consistency**: Follow established patterns visible in the codebase
3. **Integration Mindset**: Consider impact on other Omnistrate services
4. **Quality Standards**: Maintain high code quality and test coverage
5. **Documentation**: Keep documentation current with code changes

### Common Development Scenarios
- **Feature Addition**: Follow microservice patterns, add comprehensive tests
- **Bug Fixes**: Reproduce with tests, fix root cause, prevent regression
- **Refactoring**: Maintain API compatibility, update documentation
- **Performance**: Profile before optimizing, measure improvements

## 📈 Performance & Monitoring

### Key Metrics to Monitor
- Service response times
- Error rates and types
- Resource utilization
- Dependency health

### Observability Stack
- Logs: Structured logging with consistent fields
- Metrics: Prometheus-compatible metrics
- Traces: OpenTelemetry distributed tracing
- Health: HTTP health check endpoints

## 🔗 Integration Context

ctl integrates with the broader Omnistrate platform. Consider these integration points:
- Shared libraries from `commons/`
- API contracts defined in `api-design/`
- Orchestration patterns from service orchestration components
- Infrastructure dependencies and configurations

---
*This context is automatically maintained through MCP integration. Git data is refreshed daily.*
