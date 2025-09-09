package config

import (
	_ "embed"
	"fmt"
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
	clientTimeout        = "CLIENT_TIMEOUT_IN_SECONDS"
)

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
	timeoutInSeconds := GetEnvAsInteger(clientTimeout, "300")
	return time.Duration(timeoutInSeconds) * time.Second
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
