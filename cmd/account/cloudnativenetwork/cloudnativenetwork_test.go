package cloudnativenetwork

import (
	"io"
	"os"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudNativeNetworkCommandStructure(t *testing.T) {
	// Verify the top-level cloud-native-network command has the expected subcommands
	require.NotNil(t, Cmd)
	assert.Equal(t, "cloud-native-network [operation] [flags]", Cmd.Use)

	subCmds := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	assert.True(t, subCmds["list"], "expected list subcommand")
	assert.True(t, subCmds["sync"], "expected sync subcommand")
	assert.True(t, subCmds["import"], "expected import subcommand")
	assert.True(t, subCmds["remove"], "expected remove subcommand")
	assert.True(t, subCmds["host-cluster"], "expected host-cluster subcommand")
	assert.True(t, subCmds["deployment-cell"], "expected deployment-cell subcommand")
}

func TestRemoveCommandRequiresRegionAndNetworkIDFlags(t *testing.T) {
	removeCmd := findSubCommand(t, Cmd, "remove")
	require.NotNil(t, removeCmd.Flags().Lookup("region"))
	require.NotNil(t, removeCmd.Flags().Lookup("network-id"))
}

func TestImportCommandRequiresRegionAndNetworkIDFlags(t *testing.T) {
	importCmd := findSubCommand(t, Cmd, "import")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
}

func TestSyncCommandSupportsRegionAndNetworkIDFlags(t *testing.T) {
	syncCmd := findSubCommand(t, Cmd, "sync")
	require.NotNil(t, syncCmd.Flags().Lookup("region"))
	require.NotNil(t, syncCmd.Flags().Lookup("network-id"))
	require.NotNil(t, syncCmd.Flags().Lookup("network"))
	includeHostClustersFlag := syncCmd.Flags().Lookup("include-host-clusters")
	require.NotNil(t, includeHostClustersFlag)
	assert.Equal(t, "bool", includeHostClustersFlag.Value.Type())
	assert.Equal(t, "false", includeHostClustersFlag.DefValue)
}

func TestHostClusterImportCommandRequiresFlags(t *testing.T) {
	hostClusterCmd := findSubCommand(t, Cmd, "host-cluster")
	importCmd := findSubCommand(t, hostClusterCmd, "import")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
	require.NotNil(t, importCmd.Flags().Lookup("host-cluster-name"))
}

func TestValidateHostClusterImportFlags(t *testing.T) {
	require.NoError(t, validateHostClusterImportFlags("us-east-1", "vpc-abc123", "customer-eks"))

	err := validateHostClusterImportFlags("", "vpc-abc123", "customer-eks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region cannot be empty")

	err = validateHostClusterImportFlags("us-east-1", "", "customer-eks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id cannot be empty")

	err = validateHostClusterImportFlags("us-east-1", "vpc-abc123", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host-cluster-name cannot be empty")
}

func TestImportTargetsFromFlags(t *testing.T) {
	targets, err := importTargetsFromFlags("us-east-1", []string{"vpc-abc123", "vpc-def456"})
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Equal(t, "vpc-abc123", targets[0].NetworkID)
	assert.Equal(t, "us-east-1", targets[1].Region)
	assert.Equal(t, "vpc-def456", targets[1].NetworkID)
}

func TestImportTargetsFromFlagsRejectsMissingRegion(t *testing.T) {
	_, err := importTargetsFromFlags("", []string{"vpc-abc123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region cannot be empty")
}

func TestImportTargetsFromFlagsRejectsMissingNetworkID(t *testing.T) {
	_, err := importTargetsFromFlags("us-east-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one network-id is required")
}

func TestSyncTargetsFromFlags(t *testing.T) {
	targets, err := syncTargetsFromFlags(
		[]string{"us-east-1"},
		nil,
		[]string{"us-west-2:vpc-abc123"},
	)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Empty(t, targets[0].NetworkID)
	assert.Equal(t, "us-west-2", targets[1].Region)
	assert.Equal(t, "vpc-abc123", targets[1].NetworkID)
}

func TestSyncTargetsFromFlagsWithRegionAndNetworkID(t *testing.T) {
	targets, err := syncTargetsFromFlags(
		[]string{"us-east-1"},
		[]string{"vpc-abc123", "vpc-def456"},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Equal(t, "vpc-abc123", targets[0].NetworkID)
	assert.Equal(t, "us-east-1", targets[1].Region)
	assert.Equal(t, "vpc-def456", targets[1].NetworkID)
}

func TestSyncTargetsFromFlagsRejectsNetworkIDWithoutSingleRegion(t *testing.T) {
	_, err := syncTargetsFromFlags(nil, []string{"vpc-abc123"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id requires exactly one region")

	_, err = syncTargetsFromFlags([]string{"us-east-1", "us-west-2"}, []string{"vpc-abc123"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id requires exactly one region")
}

func TestSyncTargetsFromFlagsRejectsMalformedNetwork(t *testing.T) {
	_, err := syncTargetsFromFlags(nil, nil, []string{"vpc-abc123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected region:network-id")
}

func TestFormatHostClusters(t *testing.T) {
	reason := "cluster is already managed"
	hostClusters := []openapiclientfleet.FleetAccountConfigCloudNativeNetworkHostClusterResult{
		{Name: "eligible-cluster", EligibleToImport: true},
		{Name: "ineligible-cluster", EligibleToImport: false, IneligibilityReason: &reason},
	}

	assert.Equal(t,
		"eligible-cluster - eligible for import\nineligible-cluster - not eligible for import - cluster is already managed",
		formatHostClusters(hostClusters),
	)
}

func TestFormatHostClustersUsesPlaceholderForMissingReason(t *testing.T) {
	hostClusters := []openapiclientfleet.FleetAccountConfigCloudNativeNetworkHostClusterResult{
		{Name: "ineligible-cluster", EligibleToImport: false},
	}

	assert.Equal(t, "ineligible-cluster - not eligible for import", formatHostClusters(hostClusters))
}

func TestPrintCloudNativeNetworkTableIncludesHostClustersOnePerLine(t *testing.T) {
	reason := "unsupported cluster version"
	result := &openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult{
		CloudNativeNetworks: []openapiclientfleet.FleetAccountConfigCloudNativeNetworkResult{
			{
				CloudNativeNetworkId: "vpc-123",
				Region:               "us-east-1",
				Status:               "AVAILABLE",
				HostClusters: []openapiclientfleet.FleetAccountConfigCloudNativeNetworkHostClusterResult{
					{Name: "eligible-cluster", EligibleToImport: true},
					{Name: "ineligible-cluster", EligibleToImport: false, IneligibilityReason: &reason},
				},
			},
		},
	}

	var printErr error
	output := captureStdout(t, func() {
		printErr = printCloudNativeNetworkTable(result)
	})

	require.NoError(t, printErr)
	assert.Contains(t, output, "HOST CLUSTERS")
	assert.Contains(t, output, "eligible-cluster - eligible for import\n")
	assert.Contains(t, output, "ineligible-cluster - not eligible for import - unsupported cluster version")
}

func TestPrintCloudNativeNetworkTableOmitsHostClustersWithoutClusters(t *testing.T) {
	result := &openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult{
		CloudNativeNetworks: []openapiclientfleet.FleetAccountConfigCloudNativeNetworkResult{
			{
				CloudNativeNetworkId: "vpc-123",
				Region:               "us-east-1",
				Status:               "AVAILABLE",
			},
		},
	}

	var printErr error
	output := captureStdout(t, func() {
		printErr = printCloudNativeNetworkTable(result)
	})

	require.NoError(t, printErr)
	assert.NotContains(t, output, "HOST CLUSTERS")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writePipe
	defer func() {
		os.Stdout = oldStdout
		_ = readPipe.Close()
	}()

	fn()
	require.NoError(t, writePipe.Close())

	output, err := io.ReadAll(readPipe)
	require.NoError(t, err)
	return string(output)
}

func TestDeploymentCellCommandStructure(t *testing.T) {
	deploymentCellCmd := findSubCommand(t, Cmd, "deployment-cell")
	require.NotNil(t, deploymentCellCmd)

	subCmds := make(map[string]*cobra.Command)
	for _, sub := range deploymentCellCmd.Commands() {
		subCmds[sub.Name()] = sub
	}

	importCmd := subCmds["import"]
	require.NotNil(t, importCmd, "expected import subcommand")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
	require.NotNil(t, importCmd.Flags().Lookup("name"))
}

func findSubCommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	t.Fatalf("expected %s subcommand", name)
	return nil
}
