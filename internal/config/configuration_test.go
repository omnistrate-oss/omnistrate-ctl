package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func makeJWT(exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]interface{}{"exp": exp})
	claims := base64.RawURLEncoding.EncodeToString(payload)
	return fmt.Sprintf("%s.%s.fakesig", header, claims)
}

func TestIsTokenExpired_ValidToken(t *testing.T) {
	token := makeJWT(time.Now().Add(1 * time.Hour).Unix())
	assert.False(t, IsTokenExpired(token, 30*time.Second))
}

func TestIsTokenExpired_ExpiredToken(t *testing.T) {
	token := makeJWT(time.Now().Add(-1 * time.Hour).Unix())
	assert.True(t, IsTokenExpired(token, 0))
}

func TestIsTokenExpired_WithinMargin(t *testing.T) {
	token := makeJWT(time.Now().Add(10 * time.Second).Unix())
	assert.True(t, IsTokenExpired(token, 30*time.Second))
}

func TestIsTokenExpired_MalformedToken(t *testing.T) {
	assert.True(t, IsTokenExpired("not-a-jwt", 0))
	assert.True(t, IsTokenExpired("a.b", 0))
	assert.True(t, IsTokenExpired("", 0))
}

func TestIsTokenExpired_InvalidPayload(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`not json`))
	token := fmt.Sprintf("%s.%s.sig", header, payload)
	assert.True(t, IsTokenExpired(token, 0))
}

func TestIsTokenExpired_ZeroExp(t *testing.T) {
	token := makeJWT(0)
	assert.True(t, IsTokenExpired(token, 0))
}

func TestGetRetryWaitMin(t *testing.T) {
	d := GetRetryWaitMin()
	assert.Equal(t, 1*time.Second, d)
}

func TestGetRetryWaitMinCustom(t *testing.T) {
	t.Setenv(retryWaitMin, "5")
	d := GetRetryWaitMin()
	assert.Equal(t, 5*time.Second, d)
}

func TestGetRetryWaitMax(t *testing.T) {
	d := GetRetryWaitMax()
	assert.Equal(t, 30*time.Second, d)
}

func TestGetRetryWaitMaxCustom(t *testing.T) {
	t.Setenv(retryWaitMax, "60")
	d := GetRetryWaitMax()
	assert.Equal(t, 60*time.Second, d)
}

func TestGetRetryMax(t *testing.T) {
	n := GetRetryMax()
	assert.Equal(t, 5, n)
}

func TestGetRetryMaxCustom(t *testing.T) {
	t.Setenv(retryMax, "10")
	n := GetRetryMax()
	assert.Equal(t, 10, n)
}
