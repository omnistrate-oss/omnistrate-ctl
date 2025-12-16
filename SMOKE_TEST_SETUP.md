# Smoke Test Setup Guide

## Overview
This guide will help you set up and run smoke tests for the omnistrate-ctl project.

## Prerequisites
- Go 1.21 or higher
- Valid Omnistrate account credentials
- Access to Omnistrate platform (omnistrate.dev or custom domain)

## Quick Setup

### 1. Create Environment Configuration

Copy the example environment file and configure it with your credentials:

```bash
cp .env.test.example .env.test
```

Edit `.env.test` and set your credentials:

```bash
export TEST_EMAIL="your-email@omnistrate.com"
export TEST_PASSWORD="your-secure-password"
```

**Important:** Never commit `.env.test` to version control!

### 2. Run Smoke Tests

#### Using the Helper Script (Recommended)

```bash
# Run all deploy tests
./run-smoke-tests.sh

# Run specific test suite
./run-smoke-tests.sh ./test/smoke_test/build/compose Test_build

# Run all tests in a directory
./run-smoke-tests.sh ./test/smoke_test/deploy ""
```

#### Using Make

```bash
# Set credentials first
export TEST_EMAIL="your-email@omnistrate.com"
export TEST_PASSWORD="your-secure-password"

# Run smoke tests
make smoke-test
```

#### Manual Execution

```bash
# Source environment
source .env.test

# Run deploy tests
go test -v ./test/smoke_test/deploy -run Test_deploy

# Run build tests
go test -v ./test/smoke_test/build/compose -run Test_build

# Run specific test
go test -v ./test/smoke_test/deploy -run Test_deploy_basic
```

## Test Suites

### Deploy Tests (`./test/smoke_test/deploy`)

Tests for the `omctl deploy` command:

- `Test_deploy_basic` - Deploy all compose files
- `Test_deploy_with_instance` - Deploy with instance creation
- `Test_deploy_update_service` - Update existing services
- `Test_deploy_dry_run` - Dry-run mode validation
- `Test_deploy_invalid_file` - Error handling for invalid files
- `Test_deploy_no_file` - Missing file scenarios
- `Test_deploy_no_name` - Validation of required parameters
- `Test_deploy_output_format` - Different output formats
- `Test_deploy_multiple_environments` - Multi-environment deployments
- `Test_deploy_with_parameters` - Custom parameters
- `Test_deploy_no_description` - Optional fields


## Environment Variables

### Required
- `TEST_EMAIL` - Your Omnistrate account email
- `TEST_PASSWORD` - Your Omnistrate account password
- `ENABLE_SMOKE_TEST` - Set to `true` to enable smoke tests

### Optional
- `OMNISTRATE_ROOT_DOMAIN` - Platform domain (default: omnistrate.dev)
- `OMNISTRATE_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `OMNISTRATE_LOG_FORMAT` - Log format (pretty, json)
- `OMNISTRATE_HOST_SCHEME` - Protocol scheme (http, https)

## Troubleshooting

### Authentication Errors

```
Error: bad_request
Detail: Invalid request: wrong user email or password
```

**Solution:** Verify your credentials in `.env.test` are correct

### Tests Skipped

```
skipping smoke tests, set environment variable ENABLE_SMOKE_TEST
```

**Solution:** Ensure `ENABLE_SMOKE_TEST=true` is set in your environment

### Missing Credentials

```
TEST_EMAIL environment variable is not set
```

**Solution:** Source your `.env.test` file or set the environment variables manually

### Connection Issues

Check your `OMNISTRATE_ROOT_DOMAIN` and `OMNISTRATE_HOST_SCHEME` settings if you're having connection issues.

## Best Practices

1. **Never commit credentials** - Add `.env.test` to `.gitignore`
2. **Use separate test accounts** - Don't use production accounts for testing
3. **Clean up resources** - Tests create services/instances; verify cleanup
4. **Run tests in order** - Some tests may have dependencies
5. **Check test output** - Review logs for warnings or issues

## CI/CD Integration

For CI/CD pipelines, set environment variables using your CI system's secrets management:

```yaml
# GitHub Actions example
env:
  TEST_EMAIL: ${{ secrets.TEST_EMAIL }}
  TEST_PASSWORD: ${{ secrets.TEST_PASSWORD }}
  ENABLE_SMOKE_TEST: true
  OMNISTRATE_ROOT_DOMAIN: omnistrate.dev
```

## Getting Help

- Review test output for specific error messages
- Check the main README.md for general setup
- Ensure you're logged in: `omctl login`
- Verify your account has proper permissions on the platform
