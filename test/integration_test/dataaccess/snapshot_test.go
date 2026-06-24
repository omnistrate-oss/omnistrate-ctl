package dataaccess

import (
	"context"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestListAllSnapshotsWithOptions(t *testing.T) {
	testutils.IntegrationTest(t)

	serviceID := os.Getenv("SNAPSHOT_LIST_TEST_SERVICE_ID")
	environmentID := os.Getenv("SNAPSHOT_LIST_TEST_ENVIRONMENT_ID")
	productTierID := os.Getenv("SNAPSHOT_LIST_TEST_PRODUCT_TIER_ID")
	if serviceID == "" || environmentID == "" || productTierID == "" {
		t.Skip("set SNAPSHOT_LIST_TEST_SERVICE_ID, SNAPSHOT_LIST_TEST_ENVIRONMENT_ID, and SNAPSHOT_LIST_TEST_PRODUCT_TIER_ID")
	}

	ctx := context.TODO()
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, login.JWTToken)

	result, err := dataaccess.ListAllSnapshots(ctx, login.JWTToken, serviceID, environmentID, dataaccess.ListAllSnapshotsOptions{
		ProductTierID: productTierID,
		SnapshotType:  "ManualSnapshot",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	for _, snapshot := range result.Snapshots {
		require.Equal(t, "ManualSnapshot", snapshot.GetSnapshotType())
		require.Equal(t, productTierID, snapshot.GetProductTierId())
	}
}
