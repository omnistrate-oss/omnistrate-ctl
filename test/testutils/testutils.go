package testutils

import (
	"os"
	"testing"

	"github.com/pkg/errors"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

func Cleanup() {
	_ = os.RemoveAll(config.ConfigDir())
}

func Contains(arr []string, s string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}

func GetTestAccount() (string, string, error) {
	email := config.GetEnv("TEST_EMAIL", "not-set")
	password := config.GetEnv("TEST_PASSWORD", "")
	if email == "not-set" {
		return "", "", errors.New("TEST_EMAIL environment variable is not set. Set the environment variable to run the test")
	}
	if password == "" {
		return "", "", errors.New("TEST_PASSWORD environment variable is not set. Set the environment variable to run the test")
	}
	return email, password, nil
}

func SmokeTest(t *testing.T) {
	t.Helper()

	utils.ConfigureLoggingFromEnvOnce()

	if !config.GetEnvAsBoolean("ENABLE_SMOKE_TEST", "false") {
		t.Skip("skipping smoke tests, set environment variable ENABLE_SMOKE_TEST")
	}
}

func IntegrationTest(t *testing.T) {
	t.Helper()

	utils.ConfigureLoggingFromEnvOnce()

	if !config.GetEnvAsBoolean("ENABLE_INTEGRATION_TEST", "false") {
		t.Skip("skipping integration tests, set environment variable ENABLE_INTEGRATION_TEST")
	}
}
