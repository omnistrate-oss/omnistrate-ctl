package dataaccess

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestExecuteResourceInstanceCustomWorkflow(t *testing.T) {
	testutils.IntegrationTest(t)

	serviceID := os.Getenv("CUSTOM_WORKFLOW_TEST_SERVICE_ID")
	environmentID := os.Getenv("CUSTOM_WORKFLOW_TEST_ENVIRONMENT_ID")
	instanceID := os.Getenv("CUSTOM_WORKFLOW_TEST_INSTANCE_ID")
	resourceID := os.Getenv("CUSTOM_WORKFLOW_TEST_RESOURCE_ID")
	workflowID := os.Getenv("CUSTOM_WORKFLOW_TEST_WORKFLOW_ID")
	if serviceID == "" || environmentID == "" || instanceID == "" || resourceID == "" || workflowID == "" {
		t.Skip("set CUSTOM_WORKFLOW_TEST_SERVICE_ID, CUSTOM_WORKFLOW_TEST_ENVIRONMENT_ID, CUSTOM_WORKFLOW_TEST_INSTANCE_ID, CUSTOM_WORKFLOW_TEST_RESOURCE_ID, and CUSTOM_WORKFLOW_TEST_WORKFLOW_ID")
	}

	var requestParams map[string]any
	if params := os.Getenv("CUSTOM_WORKFLOW_TEST_PARAMS"); params != "" {
		require.NoError(t, json.Unmarshal([]byte(params), &requestParams))
	}

	ctx := context.TODO()
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, login.JWTToken)

	result, err := dataaccess.ExecuteResourceInstanceCustomWorkflow(ctx, login.JWTToken, serviceID, environmentID, instanceID, resourceID, workflowID, requestParams)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetWorkflowId())
	require.NotEmpty(t, result.GetWorkflowExecutionId())
}
