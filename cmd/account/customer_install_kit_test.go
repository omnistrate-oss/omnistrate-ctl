package account

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestCustomerInstallKitCommandArgs(t *testing.T) {
	require.NoError(t, customerInstallKitCmd.Args(customerInstallKitCmd, []string{"instance-123"}))
	require.Error(t, customerInstallKitCmd.Args(customerInstallKitCmd, []string{}))
	require.Error(t, customerInstallKitCmd.Args(customerInstallKitCmd, []string{"one", "two"}))
}

func TestRunCustomerInstallKitMissingAccountConfigID(t *testing.T) {
	originalSearchInventory := searchInventoryFn
	originalDescribeResourceInstance := describeResourceInstanceFn
	originalGetToken := getTokenWithLoginFn
	originalDryRun := os.Getenv("OMNISTRATE_DRY_RUN")
	t.Cleanup(func() {
		searchInventoryFn = originalSearchInventory
		describeResourceInstanceFn = originalDescribeResourceInstance
		getTokenWithLoginFn = originalGetToken
		_ = os.Setenv("OMNISTRATE_DRY_RUN", originalDryRun)
	})
	getTokenWithLoginFn = func() (string, error) { return "token", nil }
	require.NoError(t, os.Setenv("OMNISTRATE_DRY_RUN", "true"))

	resourceID := "r-injectedaccountconfigpt123"
	searchInventoryFn = func(context.Context, string, string) (*openapiclientfleet.SearchInventoryResult, error) {
		return &openapiclientfleet.SearchInventoryResult{
			ResourceInstanceResults: []openapiclientfleet.ResourceInstanceSearchRecord{
				{
					Id:                   "instance-123",
					ServiceId:            "svc-123",
					ServiceEnvironmentId: "env-123",
					ResourceId:           &resourceID,
				},
			},
		}, nil
	}
	describeResourceInstanceFn = func(context.Context, string, string, string, string) (*openapiclientfleet.ResourceInstance, error) {
		return &openapiclientfleet.ResourceInstance{
			ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{},
		}, nil
	}

	cmd := *customerInstallKitCmd
	cmd.SetArgs([]string{"instance-123"})
	err := runCustomerInstallKit(&cmd, []string{"instance-123"})
	require.ErrorContains(t, err, "does not expose a backing account config ID")
}

func TestRunCustomerInstallKitCreatesOutputDirectory(t *testing.T) {
	originalSearchInventory := searchInventoryFn
	originalDescribeResourceInstance := describeResourceInstanceFn
	originalDownload := downloadByocOnPremInstallKitFn
	originalMkdirAll := mkdirAllInstallKitFn
	originalWrite := writeInstallKitFileFn
	originalGetToken := getTokenWithLoginFn
	originalDryRun := os.Getenv("OMNISTRATE_DRY_RUN")
	t.Cleanup(func() {
		searchInventoryFn = originalSearchInventory
		describeResourceInstanceFn = originalDescribeResourceInstance
		downloadByocOnPremInstallKitFn = originalDownload
		mkdirAllInstallKitFn = originalMkdirAll
		writeInstallKitFileFn = originalWrite
		getTokenWithLoginFn = originalGetToken
		_ = os.Setenv("OMNISTRATE_DRY_RUN", originalDryRun)
		_ = customerInstallKitCmd.Flags().Set("output-path", "")
	})

	getTokenWithLoginFn = func() (string, error) { return "token", nil }
	require.NoError(t, os.Setenv("OMNISTRATE_DRY_RUN", "true"))

	resourceID := "r-injectedaccountconfigpt123"
	searchInventoryFn = func(context.Context, string, string) (*openapiclientfleet.SearchInventoryResult, error) {
		return &openapiclientfleet.SearchInventoryResult{
			ResourceInstanceResults: []openapiclientfleet.ResourceInstanceSearchRecord{
				{
					Id:                   "instance-123",
					ServiceId:            "svc-123",
					ServiceEnvironmentId: "env-123",
					ResourceId:           &resourceID,
				},
			},
		}, nil
	}
	describeResourceInstanceFn = func(context.Context, string, string, string, string) (*openapiclientfleet.ResourceInstance, error) {
		return &openapiclientfleet.ResourceInstance{
			ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
				ResultParams: map[string]any{
					customerAccountResultAccountIDKey: "ac-123",
				},
			},
		}, nil
	}
	downloadByocOnPremInstallKitFn = func(context.Context, string, string) ([]byte, string, error) {
		return []byte("kit"), "byoc-onprem-install-kit-ac-123.tar", nil
	}

	outputPath := filepath.Join(t.TempDir(), "nested", "kit.tar")
	cmd := *customerInstallKitCmd
	require.NoError(t, cmd.Flags().Set("output-path", outputPath))

	require.NoError(t, runCustomerInstallKit(&cmd, []string{"instance-123"}))
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Equal(t, []byte("kit"), data)
}
