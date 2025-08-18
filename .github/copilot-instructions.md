# GitHub Copilot Instructions for ctl

*Last Updated: 2025-08-18T02:26:43.217592*

## 🚀 Service Overview
**ctl** is a critical component of the Omnistrate platform.

### Repository Information
- **Remote**: https://github.com/omnistrate/ctl.git
- **Branch**: main
- **Total Commits**: 303

### Technologies Used
- Go
- Make
- GitHub Actions

### Health Status
**Status**: healthy
✅ No issues detected

## 📊 Recent Activity (Last 30 Days)
### Recent Commits
- `6bea672` Fix build context not resolved error after generating compose.yaml (#408) (Xinyi, 2025-08-14)
- `0f64a24` Fix product tier name in service plan details print (#407) (Alok Nikhil, 2025-08-14)
- `3d7774d` expose sync target and fix logs (#405) (Yuhui, 2025-08-11)
- `b065f44` Bump the gomod-updates group across 1 directory with 5 updates (#396) (dependabot[bot], 2025-08-07)
- `c1c6638` rename hash method (#404) (pberton, 2025-08-07)

### Files Recently Changed
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

## 👥 Top Contributors
- **pberton**: 111 commits
- **Xinyi**: 103 commits
- **Alok**: 27 commits
- **Yuhui**: 23 commits
- **dependabot[bot]**: 12 commits

## 📈 File Statistics
- **Total Files**: 5772
- **Go Files**: 5116
- **Test Files**: 39
- **Total Size**: 137.8 MB

## 🛠️ Development Guidelines

### Code Standards
- Follow Omnistrate platform conventions
- Use goa.design for API development (if applicable)
- Implement comprehensive error handling
- Write unit and integration tests
- Follow Go best practices

### Build Commands
```bash
make build          # Build the service
make test          # Run tests
make lint          # Code quality checks
make run           # Run locally
```

### Integration Points
This service integrates with other Omnistrate components. Check dependencies before making changes.

## 🔧 AI Development Assistance

When working on this service:
1. **Understand Context**: Review recent commits and changes
2. **Follow Patterns**: Use established code patterns from the repository
3. **Test Thoroughly**: Add appropriate tests for new functionality
4. **Document Changes**: Update relevant documentation
5. **Consider Dependencies**: Check impact on other services

### Common Tasks
- **API Changes**: Use goa.design patterns and regenerate code
- **Database**: Follow GORM patterns and create migrations
- **Testing**: Add table-driven tests and mock external dependencies
- **Deployment**: Update Docker and Kubernetes configurations as needed

## 📚 Quick Reference
- **Main Package**: `ctl`
- **Entry Point**: Check `cmd/` or `main.go`
- **Configuration**: Look for `config/` directory or environment variables
- **Tests**: `*_test.go` files throughout the codebase

## 🎯 Focus Areas
Based on recent activity, focus development efforts on:
- General maintenance and improvements

---
*This file is automatically updated with git data. For the latest information, ensure the MCP integration is running.*

[//]: # (maz+customer-hosted@omnistrate.com)
[//]: # (dumsud-Ziqjo3-fotmad)