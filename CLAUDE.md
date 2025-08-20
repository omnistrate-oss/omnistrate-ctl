# CLAUDE Development Context for ctl

*Last Updated: 2025-08-18T02:26:43.217592*

## 🎯 Service Purpose & Context
ctl is a key microservice in the Omnistrate platform ecosystem.

## 📊 Repository Metrics
- **Total Commits**: 303
- **Active Branch**: main
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
**2025-08-14** `6bea672` Fix build context not resolved error after generating compose.yaml (#408)
  *by Xinyi*

**2025-08-14** `0f64a24` Fix product tier name in service plan details print (#407)
  *by Alok Nikhil*

**2025-08-11** `3d7774d` expose sync target and fix logs (#405)
  *by Yuhui*

**2025-08-07** `b065f44` Bump the gomod-updates group across 1 directory with 5 updates (#396)
  *by dependabot[bot]*

**2025-08-07** `c1c6638` rename hash method (#404)
  *by pberton*

**2025-08-07** `b0e325b` add deployment cell sidebar (#403)
  *by Yuhui*

**2025-08-06** `8ab1614` print preview before applying (#402)
  *by Yuhui*

**2025-08-06** `2410aa9` fix ctl print (#401)
  *by Yuhui*

**2025-08-06** `842868b` update signing logic (#400)
  *by pberton*

**2025-08-05** `af42fd2` Add Clean Live Log Streaming and Syntax Highlighting to CTL TUI (#393)
  *by Mohan Dholu*

### Active Development Areas
- .github/workflows/build-docs.yaml
- .github/workflows/docs-deploy.yml
- .github/workflows/package.yml
- .github/workflows/release.yml
- Makefile
- build/Dockerfile
- build/Dockerfile.docs
- build/Dockerfile.docs.local
- cmd/alarms/alarms.go
- cmd/alarms/notificationchannel/event_history.go

## 👥 Development Team Context

### Contributors & Expertise
- **pberton**: 111 commits (High activity)
- **Xinyi**: 103 commits (High activity)
- **Alok**: 27 commits (Medium activity)
- **Yuhui**: 23 commits (Medium activity)
- **dependabot[bot]**: 12 commits (Medium activity)
- **maziar kaveh**: 11 commits (Medium activity)
- **Tomislav Simeunovic**: 6 commits (Light activity)
- **Copilot**: 3 commits (Light activity)

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
- **Test Coverage**: 39 test files
- **Code Size**: 137.8 MB
- **Go Files**: 5116 files

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
