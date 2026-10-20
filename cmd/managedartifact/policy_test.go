package managedartifact

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPolicyUpdateRespectsUserFeatureFlags(t *testing.T) {
	t.Cleanup(homedir.Reset)
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Setenv("OMNISTRATE_API_KEY", "")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
	var flags map[string]bool
	var updateCalls int
	var rejectUpdate bool
	var updatedPath string
	var updatedRequest model.UpdateManagedArtifactReleasePolicyRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/2022-09-01-00/user" {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "mock-user", "featureFlags": flags}))
			return
		}
		require.Equal(t, http.MethodPut, r.Method)
		updateCalls++
		updatedPath = r.URL.Path
		if rejectUpdate {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"name":"forbidden","message":"managed artifact release pinning is not enabled for cloud provider azure"}`))
			return
		}
		var request model.UpdateManagedArtifactReleasePolicyRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		updatedRequest = request
		require.NoError(t, json.NewEncoder(w).Encode(model.ManagedArtifactReleasePolicy{
			AutoUpgrade: request.AutoUpgrade, PreferredBundleVersion: request.PreferredBundleVersion,
		}))
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	// An unsigned local fixture, never a live credential.
	token := "mock." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":4102444800}`)) + ".mock"
	require.NoError(t, config.CreateOrUpdateAuthConfig(config.AuthConfig{Token: token}))

	for _, provider := range []struct{ cloud, flag string }{
		{"aws", "USE_SP_ECR_FOR_MANAGED_ARTIFACTS"},
		{"azure", "USE_SP_ECR_FOR_MANAGED_ARTIFACTS_AZURE"},
		{"gcp", "USE_SP_ECR_FOR_MANAGED_ARTIFACTS_GCP"},
		{"byoc-onprem", "USE_SP_ECR_FOR_MANAGED_ARTIFACTS_BYOK"},
	} {
		for _, enabled := range []bool{false, true} {
			for _, version := range []string{"", "r0000277"} {
				t.Run(provider.cloud+"/"+version+"/enabled="+map[bool]string{false: "false", true: "true"}[enabled], func(t *testing.T) {
					// AWS's flag must never authorize another cloud.
					flags = map[string]bool{"USE_SP_ECR_FOR_MANAGED_ARTIFACTS": true}
					flags[provider.flag] = enabled
					before := updateCalls
					command := &cobra.Command{Use: "update", RunE: runPolicyUpdate, SilenceErrors: true, SilenceUsage: true}
					addPolicyScopeFlags(command)
					command.Flags().Bool("auto-upgrade", false, "Automatically adopt new releases")
					command.Flags().String("preferred-bundle-version", version, "Release to pin")
					command.Flags().String(common.OutputFlag, "json", "Output format")
					command.SetArgs([]string{"--environment-type", "global", "--cloud-provider", provider.cloud})
					err := command.ExecuteContext(t.Context())
					if enabled {
						require.NoError(t, err)
						require.Equal(t, before+1, updateCalls)
						require.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/GLOBAL/"+provider.cloud, updatedPath)
						require.False(t, updatedRequest.AutoUpgrade)
						require.Equal(t, version, updatedRequest.PreferredBundleVersion)
					} else {
						require.ErrorContains(t, err, provider.flag)
						require.Equal(t, before, updateCalls)
					}
				})
			}
		}
	}
	for _, scenario := range []string{"missing flags", "backend revoked", "clear pin", "nebius", "oci"} {
		t.Run(scenario, func(t *testing.T) {
			flags = nil
			rejectUpdate = scenario == "backend revoked"
			if rejectUpdate {
				flags = map[string]bool{"USE_SP_ECR_FOR_MANAGED_ARTIFACTS_AZURE": true}
			}
			before := updateCalls
			command := &cobra.Command{Use: "update", RunE: runPolicyUpdate, SilenceErrors: true, SilenceUsage: true}
			addPolicyScopeFlags(command)
			command.Flags().Bool("auto-upgrade", scenario == "clear pin", "Automatically adopt new releases")
			command.Flags().String("preferred-bundle-version", "", "Release to pin")
			command.Flags().String(common.OutputFlag, "json", "Output format")
			cloud := "azure"
			if scenario == "nebius" || scenario == "oci" {
				cloud = scenario
			}
			command.SetArgs([]string{"--environment-type", "global", "--cloud-provider", cloud})
			err := command.ExecuteContext(t.Context())
			switch scenario {
			case "missing flags":
				require.ErrorContains(t, err, "USE_SP_ECR_FOR_MANAGED_ARTIFACTS_AZURE")
				require.Equal(t, before, updateCalls)
			case "backend revoked":
				require.ErrorContains(t, err, "pinning is not enabled")
				require.Equal(t, before+1, updateCalls)
			case "clear pin":
				require.NoError(t, err)
				require.Equal(t, before+1, updateCalls)
				require.True(t, updatedRequest.AutoUpgrade)
				require.Empty(t, updatedRequest.PreferredBundleVersion)
			default:
				require.ErrorContains(t, err, "pinning is not available for cloud provider "+cloud)
				require.Equal(t, before, updateCalls)
			}
		})
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
