package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetLogLevel(t *testing.T) {
	logLevel := GetLogLevel()
	assert.Equal(t, "info", logLevel)
}

func TestGetLogLevelCustom(t *testing.T) {
	t.Setenv(logLevel, "debug")
	logLevel := GetLogLevel()
	assert.Equal(t, "debug", logLevel)
}

func TestGetLogFormat(t *testing.T) {
	logFormat := GetLogFormat()
	assert.Equal(t, "pretty", logFormat)
}

func TestGetLogFormatCustom(t *testing.T) {
	t.Setenv(logFormat, "json")
	logFormat := GetLogFormat()
	assert.Equal(t, "json", logFormat)
}

func TestGetHost(t *testing.T) {
	host := GetHost()
	assert.Equal(t, "api.omnistrate.cloud", host)
}

func TestGetHostCustom(t *testing.T) {
	t.Setenv(omnistrateHost, "example.com")
	host := GetHost()
	assert.Equal(t, "example.com", host)
}

func TestGetRootDomain(t *testing.T) {
	rootDomain := GetRootDomain()
	assert.Equal(t, "omnistrate.cloud", rootDomain)
}

func TestGetRootDomainCustom(t *testing.T) {
	t.Setenv(omnistrateRootDomain, "example.com")
	rootDomain := GetRootDomain()
	assert.Equal(t, "example.com", rootDomain)
}

func TestGetHostScheme(t *testing.T) {
	hostScheme := GetHostScheme()
	assert.Equal(t, "https", hostScheme)
}

func TestGetHostSchemeCustom(t *testing.T) {
	t.Setenv(omnistrateHostSchema, "http")
	hostScheme := GetHostScheme()
	assert.Equal(t, "http", hostScheme)
}

func TestGetDebug(t *testing.T) {
	debug := IsDebugLogLevel()
	assert.False(t, debug)
}

func TestGetDebugTrue(t *testing.T) {
	t.Setenv(logLevel, "debug")
	debug := IsDebugLogLevel()
	assert.True(t, debug)
}

func TestGetClientTimeout(t *testing.T) {
	clientTimeout := GetClientTimeout()
	assert.Equal(t, time.Duration(300000000000), clientTimeout)
}

func TestGetClientTimeoutOverride(t *testing.T) {
	t.Setenv(clientTimeout, "1")
	clientTimeout := GetClientTimeout()
	assert.Equal(t, time.Duration(1)*time.Second, clientTimeout)
}

func TestDryRun(t *testing.T) {
	t.Setenv(dryRunEnv, "true")
	assert.True(t, IsDryRun(), "DryRun should be true for tests")
}

func TestDryRunModify(t *testing.T) {
	t.Setenv(dryRunEnv, "false")
	assert.False(t, IsDryRun(), "DryRun should be false")
	t.Setenv(dryRunEnv, "true")
	assert.True(t, IsDryRun(), "DryRun should be true")
}
func TestGetLlmsTxtURL(t *testing.T) {
	url := GetLlmsTxtURL()
	assert.Equal(t, "https://docs.omnistrate.com/llms.txt", url)
}

func TestGetLlmsTxtURLCustom(t *testing.T) {
	t.Setenv(omnistrateDocsDomain, "custom.example.com")
	url := GetLlmsTxtURL()
	assert.Equal(t, "https://custom.example.com/llms.txt", url)
}
