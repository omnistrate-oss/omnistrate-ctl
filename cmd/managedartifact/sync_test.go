package managedartifact

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestSyncListRejectsInvalidDestinationFiltersBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name, registry, account, want string
	}{
		{"unknown registry", "docker", "", "invalid registry type"},
		{"short account", "PUBLIC_ECR", "123", "exactly 12 digits"},
		{"non-numeric account", "PRIVATE_ECR", "12345678901x", "exactly 12 digits"},
		{"padded account", "", " 123456789012 ", "exactly 12 digits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, syncListCmd.Flags().Set("registry-type", tc.registry))
			require.NoError(t, syncListCmd.Flags().Set("destination-account-id", tc.account))
			require.ErrorContains(t, runSyncList(syncListCmd, nil), tc.want)
		})
	}
}

func TestManagedArtifactPublicSyncCommands(t *testing.T) {
	t.Cleanup(homedir.Reset)
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Setenv("OMNISTRATE_API_KEY", "")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
	digest := "sha256:" + strings.Repeat("a", 64)
	record := model.ManagedArtifactSync{
		ID: "sppap-unassigned", RegistryType: "PUBLIC_ECR", BundleVersion: "r0000020", Status: "READY",
		Destination:   &model.ManagedArtifactDestination{CloudProvider: "aws", AccountID: "123456789012", Registry: "public.ecr.aws/example"},
		ArtifactCount: 1, CompletedArtifactCount: 1,
		Artifacts: []model.ManagedArtifactSyncArtifact{{
			AmenityName: "dataplane-agent", ArtifactKey: "helm-chart/dataplane-agent-chart", Version: "v1.2.3", Status: "READY",
			SourceRef: "oci://ghcr.io/example/chart:v1.2.3", SourceDigest: digest,
			DestinationRef: "oci://public.ecr.aws/example/artifacts/chart:v1.2.3", DestinationDigest: digest,
		}},
	}
	var capturedQuery url.Values
	var listOverride *model.ManagedArtifactSyncList
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/2022-09-01-00/user":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"id": "mock-user"}))
		case "/2022-09-01-00/managed-artifact/syncs":
			capturedQuery = r.URL.Query()
			result := model.ManagedArtifactSyncList{Syncs: []model.ManagedArtifactSync{record}}
			if listOverride != nil {
				result = *listOverride
			}
			require.NoError(t, json.NewEncoder(w).Encode(result))
		case "/2022-09-01-00/managed-artifact/syncs/sppap-unassigned":
			require.NoError(t, json.NewEncoder(w).Encode(record))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	// An unsigned local fixture, never a real credential or live authentication.
	token := "mock." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":4102444800}`)) + ".mock"
	require.NoError(t, config.CreateOrUpdateAuthConfig(config.AuthConfig{Token: token}))

	newCommand := func(t *testing.T, prototype *cobra.Command, output string) *cobra.Command {
		t.Helper()
		command := &cobra.Command{Use: prototype.Use, RunE: prototype.RunE, SilenceErrors: true, SilenceUsage: true}
		prototype.Flags().VisitAll(func(flag *pflag.Flag) {
			switch flag.Value.Type() {
			case "string":
				command.Flags().String(flag.Name, flag.DefValue, flag.Usage)
			case "int":
				defaultValue, err := strconv.Atoi(flag.DefValue)
				require.NoError(t, err)
				command.Flags().Int(flag.Name, defaultValue, flag.Usage)
			}
		})
		command.Flags().String(common.OutputFlag, output, "Output format")
		return command
	}

	for _, prototype := range []*cobra.Command{syncListCmd, syncDescribeCmd} {
		for _, output := range []string{"json", "table"} {
			t.Run(prototype.Name()+"/"+output, func(t *testing.T) {
				command := newCommand(t, prototype, output)
				if prototype == syncListCmd {
					command.SetArgs([]string{"--registry-type", "public_ecr", "--destination-account-id", "123456789012", "--target-id", "hc-executor"})
				} else {
					command.SetArgs([]string{"--id", record.ID})
				}
				utils.LastPrintedString = ""
				require.NoError(t, command.ExecuteContext(t.Context()))
				if prototype == syncListCmd {
					require.Equal(t, "PUBLIC_ECR", capturedQuery.Get("registryType"))
					require.Equal(t, "123456789012", capturedQuery.Get("destinationAccountId"))
					require.Equal(t, "hc-executor", capturedQuery.Get("targetId"))
					require.Contains(t, utils.LastPrintedString, "public.ecr.aws/example")
				} else {
					require.Contains(t, utils.LastPrintedString, record.Artifacts[0].DestinationRef)
					if output == "json" {
						require.Contains(t, utils.LastPrintedString, digest)
						require.Contains(t, utils.LastPrintedString, record.Artifacts[0].SourceRef)
					} else {
						require.NotContains(t, utils.LastPrintedString, digest)
						require.NotContains(t, utils.LastPrintedString, record.Artifacts[0].SourceRef)
					}
				}
				if output == "json" {
					require.NotContains(t, utils.LastPrintedString, `"target"`, "unassigned targets must remain absent")
					require.Contains(t, utils.LastPrintedString, `"destination"`)
				}
			})
		}
	}
	// With no new filters, older backends must receive the same private request.
	legacy := record
	legacy.ID, legacy.RegistryType, legacy.Destination = "spabs-legacy", "", nil
	legacy.Target = &model.ManagedArtifactTarget{ID: "hc-private", CloudProvider: "aws", AccountID: "123456789012"}
	record = legacy
	for _, output := range []string{"json", "table"} {
		t.Run("legacy list/"+output, func(t *testing.T) {
			command := newCommand(t, syncListCmd, output)
			require.NoError(t, command.ExecuteContext(t.Context()))
			require.False(t, capturedQuery.Has("registryType"))
			require.False(t, capturedQuery.Has("destinationAccountId"))
			require.Contains(t, utils.LastPrintedString, "hc-private")
		})
	}
	private := legacy
	private.RegistryType = "PRIVATE_ECR"
	private.Destination = &model.ManagedArtifactDestination{AccountID: "123456789012"}
	public := legacy
	public.ID, public.RegistryType = "sppap-filtered", "PUBLIC_ECR"
	public.Destination = &model.ManagedArtifactDestination{AccountID: "123456789012"}
	future := public
	future.RegistryType = "FUTURE_REGISTRY"
	for _, tc := range []struct {
		name     string
		registry string
		account  string
		records  []model.ManagedArtifactSync
		wantErr  bool
	}{
		{"old backend public query", "PUBLIC_ECR", "", []model.ManagedArtifactSync{legacy}, true},
		{"old backend explicit private query", "PRIVATE_ECR", "", []model.ManagedArtifactSync{legacy}, true},
		{"old backend account-only query", "", "123456789012", []model.ManagedArtifactSync{legacy}, true},
		{"wrong registry", "PUBLIC_ECR", "", []model.ManagedArtifactSync{private}, true},
		{"mixed public and private results", "PUBLIC_ECR", "", []model.ManagedArtifactSync{public, private}, true},
		{"missing destination", "PUBLIC_ECR", "123456789012", []model.ManagedArtifactSync{{ID: "sppap-missing", RegistryType: "PUBLIC_ECR"}}, true},
		{"wrong account", "PUBLIC_ECR", "210987654321", []model.ManagedArtifactSync{public}, true},
		{"public query honored", "PUBLIC_ECR", "123456789012", []model.ManagedArtifactSync{public}, false},
		{"private account query honored", "", "123456789012", []model.ManagedArtifactSync{private}, false},
		{"implicit private query rejects public", "", "123456789012", []model.ManagedArtifactSync{public}, true},
		{"omitted filters preserve future values", "", "", []model.ManagedArtifactSync{future}, false},
		{"empty page does not prove support", "PUBLIC_ECR", "123456789012", nil, false},
	} {
		for _, output := range []string{"json", "table"} {
			t.Run(tc.name+"/"+output, func(t *testing.T) {
				listOverride = &model.ManagedArtifactSyncList{Syncs: tc.records}
				command := newCommand(t, syncListCmd, output)
				command.SetArgs([]string{"--registry-type", tc.registry, "--destination-account-id", tc.account})
				utils.LastPrintedString = ""
				err := command.ExecuteContext(t.Context())
				if tc.wantErr {
					require.ErrorContains(t, err, "backend did not honor")
					require.Empty(t, utils.LastPrintedString, "mismatched records must not be displayed")
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestManagedArtifactSyncSummarySupportsLegacyAndPublicDestinations(t *testing.T) {
	for _, tc := range []struct {
		name                              string
		sync                              model.ManagedArtifactSync
		registry, account, target, region string
	}{
		{"legacy private", model.ManagedArtifactSync{Target: &model.ManagedArtifactTarget{ID: "hc-private", CloudProvider: "aws", Region: "us-west-2", AccountID: "123456789012"}}, "PRIVATE_ECR", "123456789012", "hc-private", "us-west-2"},
		{"public unassigned", model.ManagedArtifactSync{RegistryType: "PUBLIC_ECR", Destination: &model.ManagedArtifactDestination{CloudProvider: "aws", AccountID: "210987654321"}}, "PUBLIC_ECR", "210987654321", "", ""},
		{"public executor is not destination", model.ManagedArtifactSync{RegistryType: "PUBLIC_ECR", Target: &model.ManagedArtifactTarget{ID: "hc-executor", CloudProvider: "gcp", Region: "us-central1", AccountID: "123456789012"}, Destination: &model.ManagedArtifactDestination{CloudProvider: "aws", AccountID: "210987654321"}}, "PUBLIC_ECR", "210987654321", "hc-executor", "us-central1"},
		{"future registry preserves destination", model.ManagedArtifactSync{RegistryType: "FUTURE_REGISTRY", Destination: &model.ManagedArtifactDestination{AccountID: "210987654321"}}, "FUTURE_REGISTRY", "210987654321", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := syncSummary(tc.sync)
			require.Equal(t, tc.registry, row.RegistryType)
			require.Equal(t, tc.account, row.DestinationAccountID)
			require.Equal(t, tc.target, row.ExecutionTargetID)
			require.Equal(t, tc.region, row.ExecutionRegion)
		})
	}
}
