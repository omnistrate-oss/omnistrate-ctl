package config

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	CommitID  string
	Version   string
	Timestamp string
)

const (
	dryRunEnv            = "OMNISTRATE_DRY_RUN"
	logLevel             = "OMNISTRATE_LOG_LEVEL"
	logFormat            = "OMNISTRATE_LOG_FORMAT_LEVEL"
	omnistrateHost       = "OMNISTRATE_HOST"
	omnistrateRootDomain = "OMNISTRATE_ROOT_DOMAIN"
	omnistrateHostSchema = "OMNISTRATE_HOST_SCHEME"
	omnistrateDocsDomain = "OMNISTRATE_DOCS_DOMAIN"
	defaultRootDomain    = "omnistrate.cloud"
	clientTimeoutOld     = "CLIENT_TIMEOUT_IN_SECONDS"
	clientTimeout        = "OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS"
	retryWaitMin         = "OMNISTRATE_RETRY_WAIT_MIN_IN_SECONDS"
	retryWaitMax         = "OMNISTRATE_RETRY_WAIT_MAX_IN_SECONDS"
	retryMax             = "OMNISTRATE_RETRY_MAX"
)

func GetComposeSpecUrl() string {
	return fmt.Sprintf("https://%s/spec-guides/compose-spec/index.md", GetOmnistrateDocsDomain())
}

func GetPlanSpecUrl() string {
	return fmt.Sprintf("https://%s/spec-guides/plan-spec/index.md", GetOmnistrateDocsDomain())
}

func GetLlmsTxtURL() string {
	return fmt.Sprintf("https://%s/llms.txt", GetOmnistrateDocsDomain())
}

func GetOmnistrateDocsDomain() string {
	return GetEnv(omnistrateDocsDomain, "docs.omnistrate.com")
}

// GetToken returns the authentication token for current user
func GetToken() (string, error) {
	authConfig, err := LookupAuthConfig()
	if err != nil {
		return "", err
	}

	return authConfig.Token, nil
}

// GetRefreshToken returns the stored refresh token.
func GetRefreshToken() (string, error) {
	authConfig, err := LookupAuthConfig()
	if err != nil {
		return "", err
	}
	if authConfig.RefreshToken == "" {
		return "", ErrAuthConfigNotFound
	}
	return authConfig.RefreshToken, nil
}

// IsTokenExpired parses the JWT's exp claim and returns true if the token
// has expired or will expire within the given margin. This avoids a network
// round-trip to validate the token when we can tell locally it's stale.
func IsTokenExpired(token string, margin time.Duration) bool {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return true
	}

	// JWT payload is base64url-encoded (no padding)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return true
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return true
	}

	return time.Now().Add(margin).Unix() >= claims.Exp
}

func GetIndexCacheTTL() time.Duration {
	ttlInSeconds := GetEnvAsInteger("OMNISTRATE_INDEX_CACHE_TTL", "3600")
	return time.Duration(ttlInSeconds) * time.Second
}

// GetSearchTimestampFilePath returns the path to the timestamp file used to track the last update time of the search index
func GetSearchTimestampFilePath() string {
	indexDir := GetSearchIndexDir()
	indexName := GetSearchIndexName()
	timestampFileName := indexName + ".timestamp"
	return filepath.Join(indexDir, timestampFileName)
}

func GetSearchIndexPath() string {
	return filepath.Join(GetSearchIndexDir(), GetSearchIndexName())
}

// GetSearchIndexDir returns the directory where search index files are stored
func GetSearchIndexDir() string {
	return GetEnv("OMNISTRATE_SEARCH_INDEX_DIR", filepath.Join(ConfigDir(), "search"))
}

// GetSearchIndexName returns the name of the search index file
func GetSearchIndexName() string {
	return GetEnv("OMNISTRATE_SEARCH_INDEX_NAME", "search_index.bleve")
}

// GetHost returns the host of the Omnistrate server
func GetHost() string {
	return GetEnv(omnistrateHost, "api"+"."+GetRootDomain())
}

// GetRootDomain returns the root domain of the Omnistrate server
func GetRootDomain() string {
	return GetEnv(omnistrateRootDomain, defaultRootDomain)
}

// GetHostScheme returns the scheme of the Omnistrate server
func GetHostScheme() string {
	return GetEnv(omnistrateHostSchema, "https")
}

func GetLogLevel() string {
	return GetEnv(logLevel, "info")
}

func IsDebugLogLevel() bool {
	return strings.EqualFold(GetLogLevel(), "debug")
}

func GetLogFormat() string {
	return GetEnv(logFormat, "pretty")
}

//go:embed public_key.pem
var publicKey []byte

// GetDefaultServiceAuthPublicKey returns the default public key for environment creation
func GetDefaultServiceAuthPublicKey() string {
	return string(publicKey)
}

func IsProd() bool {
	return GetRootDomain() == defaultRootDomain
}

func IsDryRun() bool {
	return GetEnvAsBoolean(dryRunEnv, "false")
}

func GetClientTimeout() time.Duration {
	// Prefer OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS, fall back to CLIENT_TIMEOUT_IN_SECONDS
	if v := os.Getenv(clientTimeout); v != "" {
		timeoutInSeconds := GetEnvAsInteger(clientTimeout, "300")
		return time.Duration(timeoutInSeconds) * time.Second
	}
	timeoutInSeconds := GetEnvAsInteger(clientTimeoutOld, "300")
	return time.Duration(timeoutInSeconds) * time.Second
}

// GetRetryWaitMin returns the minimum wait time between retries
func GetRetryWaitMin() time.Duration {
	waitInSeconds := GetEnvAsInteger(retryWaitMin, "5")
	return time.Duration(waitInSeconds) * time.Second
}

// GetRetryWaitMax returns the maximum wait time between retries
func GetRetryWaitMax() time.Duration {
	waitInSeconds := GetEnvAsInteger(retryWaitMax, "30")
	return time.Duration(waitInSeconds) * time.Second
}

// GetRetryMax returns the maximum number of retries
func GetRetryMax() int {
	return GetEnvAsInteger(retryMax, "5")
}

// GetUserAgent returns the User-Agent string for HTTP requests
func GetUserAgent() string {
	if Version == "" {
		return "omnistrate-ctl/unknown"
	}
	return "omnistrate-ctl/" + Version
}

func CleanupArgsAndFlags(cmd *cobra.Command, args *[]string) {
	// Clean up flags
	cmd.Flags().VisitAll(
		func(f *pflag.Flag) {
			_ = cmd.Flags().Set(f.Name, f.DefValue)
		})

	// Clean up arguments by resetting the slice to nil or an empty slice
	*args = nil
}
