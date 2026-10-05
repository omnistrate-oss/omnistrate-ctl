package managedartifact

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPolicyUpdateRejectsNonAWSPinningBeforeAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unsupported pinning must be rejected before any API request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	for _, provider := range []string{"azure", "gcp", "nebius", "oci", "byoc-onprem"} {
		for _, bundleVersion := range []string{"", "r0000020"} {
			name := provider + "/implicit pin"
			if bundleVersion != "" {
				name = provider + "/explicit pin"
			}
			t.Run(name, func(t *testing.T) {
				require.NoError(t, policyUpdateCmd.Flags().Set("environment-type", "dev"))
				require.NoError(t, policyUpdateCmd.Flags().Set("cloud-provider", provider))
				require.NoError(t, policyUpdateCmd.Flags().Set("auto-upgrade", "false"))
				require.NoError(t, policyUpdateCmd.Flags().Set("preferred-bundle-version", bundleVersion))

				err := runPolicyUpdate(policyUpdateCmd, nil)
				require.EqualError(t, err, "--auto-upgrade=false is supported only for aws")
			})
		}
	}
}

func TestPolicyUpdateRejectsUnsupportedOutputBeforeAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unsupported output must be rejected before authentication or policy mutation")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_API_KEY", "mock-api-key")

	command := &cobra.Command{Use: "update", RunE: runPolicyUpdate, SilenceErrors: true, SilenceUsage: true}
	addPolicyScopeFlags(command)
	command.Flags().Bool("auto-upgrade", false, "Automatically adopt new releases")
	command.Flags().String("preferred-bundle-version", "", "Release to pin")
	command.Flags().String(common.OutputFlag, "table", "Output format")

	for _, autoUpgrade := range []string{"true", "false"} {
		t.Run("auto-upgrade="+autoUpgrade, func(t *testing.T) {
			command.SetArgs([]string{"--environment-type", "dev",
				"--cloud-provider", "aws", "--auto-upgrade=" + autoUpgrade, "--output", "yaml"})
			require.EqualError(t, command.ExecuteContext(t.Context()), "unsupported output format: yaml")
		})
	}
}
