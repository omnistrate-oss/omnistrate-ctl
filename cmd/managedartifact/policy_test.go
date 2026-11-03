package managedartifact

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPolicyUpdateDoesNotGateProviderOrFeatureAccess(t *testing.T) {
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
	var currentPolicy model.ManagedArtifactReleasePolicy
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/2022-09-01-00/user" {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "mock-user", "featureFlags": flags}))
			return
		}
		if r.Method == http.MethodGet {
			require.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/GLOBAL/byoc-onprem", r.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(currentPolicy))
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
		currentPolicy = model.ManagedArtifactReleasePolicy{
			AutoUpgrade: request.AutoUpgrade, EffectiveBundleVersion: "r0000295",
		}
		if !request.AutoUpgrade {
			if request.PreferredBundleVersion != "" {
				currentPolicy.EffectiveBundleVersion = request.PreferredBundleVersion
			}
			currentPolicy.PreferredBundleVersion = currentPolicy.EffectiveBundleVersion
		}
		require.NoError(t, json.NewEncoder(w).Encode(currentPolicy))
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	// An unsigned local fixture, never a live credential.
	token := "mock." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":4102444800}`)) + ".mock"
	require.NoError(t, config.CreateOrUpdateAuthConfig(config.AuthConfig{Token: token}))

	newCommand := func(cloud, version string, autoUpgrade bool) *cobra.Command {
		command := &cobra.Command{Use: "update", RunE: runPolicyUpdate, SilenceErrors: true, SilenceUsage: true}
		addPolicyScopeFlags(command)
		command.Flags().Bool("auto-upgrade", false, "Automatically adopt new releases")
		command.Flags().String("preferred-bundle-version", "", "Release to pin")
		command.Flags().String(common.OutputFlag, "json", "Output format")
		args := []string{"--environment-type", "global", "--cloud-provider", cloud, "--auto-upgrade=" + strconv.FormatBool(autoUpgrade)}
		if version != "" {
			args = append(args, "--preferred-bundle-version", version)
		}
		command.SetArgs(args)
		return command
	}

	flags = nil
	for _, provider := range []string{"aws", "azure", "gcp", "nebius", "oci", "byoc-onprem"} {
		for _, version := range []string{"", "r0000277"} {
			t.Run(provider+"/"+version+"/missing feature flags", func(t *testing.T) {
				before := updateCalls
				require.NoError(t, newCommand(provider, version, false).ExecuteContext(t.Context()))
				require.Equal(t, before+1, updateCalls)
				require.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/GLOBAL/"+provider, updatedPath)
				require.False(t, updatedRequest.AutoUpgrade)
				require.Equal(t, version, updatedRequest.PreferredBundleVersion)
			})
		}
	}

	t.Run("disabled feature flag is ignored", func(t *testing.T) {
		flags = map[string]bool{"USE_SP_ECR_FOR_MANAGED_ARTIFACTS_AZURE": false}
		before := updateCalls
		require.NoError(t, newCommand("azure", "r0000277", false).ExecuteContext(t.Context()))
		require.Equal(t, before+1, updateCalls)
	})

	t.Run("backend rejection remains authoritative", func(t *testing.T) {
		flags = nil
		rejectUpdate = true
		before := updateCalls
		err := newCommand("azure", "r0000277", false).ExecuteContext(t.Context())
		require.ErrorContains(t, err, "pinning is not enabled")
		require.Equal(t, before+1, updateCalls)
		rejectUpdate = false
	})

	t.Run("clear pin", func(t *testing.T) {
		flags = nil
		before := updateCalls
		require.NoError(t, newCommand("azure", "", true).ExecuteContext(t.Context()))
		require.Equal(t, before+1, updateCalls)
		require.True(t, updatedRequest.AutoUpgrade)
		require.Empty(t, updatedRequest.PreferredBundleVersion)
	})

	t.Run("BYOK bundle changes and auto-upgrade restoration", func(t *testing.T) {
		flags = nil
		rejectUpdate = false
		for _, step := range []struct {
			name, version, effective string
			autoUpgrade              bool
		}{
			{name: "pin previous release", version: "r0000294", effective: "r0000294"},
			{name: "change pinned release", version: "r0000293", effective: "r0000293"},
			{name: "restore auto-upgrade", effective: "r0000295", autoUpgrade: true},
		} {
			t.Run(step.name, func(t *testing.T) {
				before := updateCalls
				require.NoError(t, newCommand("byoc-onprem", step.version, step.autoUpgrade).ExecuteContext(t.Context()))
				require.Equal(t, before+1, updateCalls)
				require.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/GLOBAL/byoc-onprem", updatedPath)
				policy, err := dataaccess.DescribeManagedArtifactReleasePolicy(t.Context(), token, "GLOBAL", "byoc-onprem")
				require.NoError(t, err)
				require.Equal(t, step.autoUpgrade, policy.AutoUpgrade)
				require.Equal(t, step.version, policy.PreferredBundleVersion)
				require.Equal(t, step.effective, policy.EffectiveBundleVersion)
			})
		}
	})
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
