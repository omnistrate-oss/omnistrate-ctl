package common

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDeploymentCellAccessReturnsApprovedRequest(t *testing.T) {
	var output bytes.Buffer
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", "", deploymentCellAccessDependencies{
			create: func(_ context.Context, token string, input dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				require.Equal(t, "token", token)
				require.Equal(t, "hc-test", input.DeploymentCellID)
				require.Equal(t, "cluster-admin", input.Role)
				require.Nil(t, input.Reason)
				return &model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				t.Fatal("approved request must not be polled")
				return nil, nil
			},
			output: &output,
		})

	require.NoError(t, err)
	require.Equal(t, "ar-approved", request.ID)
	require.Contains(t, output.String(), "Deployment cell hc-test: access request ar-approved is APPROVED.")
}

func TestEnsureDeploymentCellAccessWaitsForManualApproval(t *testing.T) {
	var output bytes.Buffer
	describeCalls := 0
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", " Investigate incident ", deploymentCellAccessDependencies{
			create: func(_ context.Context, _ string, input dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				require.NotNil(t, input.Reason)
				require.Equal(t, "Investigate incident", *input.Reason)
				return &model.AccessRequest{ID: "ar-pending", Status: "PENDING"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				describeCalls++
				if describeCalls == 1 {
					return &model.AccessRequest{ID: "ar-pending", Status: "APPROVING"}, nil
				}
				return &model.AccessRequest{ID: "ar-pending", Status: "APPROVED"}, nil
			},
			output: &output,
			configuredApprovers: func(context.Context, string, string) ([]string, error) {
				return []string{"first@example.com", "second@example.com"}, nil
			},
			pollInterval: time.Millisecond,
			autoTimeout:  time.Second,
		})

	require.NoError(t, err)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 2, describeCalls)
	require.Contains(t, output.String(), "Deployment cell hc-test: access request ar-pending is PENDING")
	require.Contains(t, output.String(), "Deployment cell hc-test: access request ar-pending is APPROVING")
	require.Contains(t, output.String(), "Deployment cell hc-test: access request ar-pending is APPROVED")
	require.Contains(t, output.String(), "Configured approvers: first@example.com, second@example.com")
	require.NotContains(t, output.String(), "http")
}

func TestEnsureDeploymentCellAccessReportsAutomaticApprovalProgress(t *testing.T) {
	var output bytes.Buffer
	describeCalls := 0
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-provisioner", "cluster-admin", "debug instance", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-auto", Status: "APPROVING"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				describeCalls++
				return &model.AccessRequest{ID: "ar-auto", Status: "APPROVED"}, nil
			},
			output:       &output,
			pollInterval: time.Millisecond,
			autoTimeout:  time.Second,
		})

	require.NoError(t, err)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 1, describeCalls)
	require.Contains(t, output.String(), "Deployment cell hc-provisioner: access request ar-auto is APPROVING")
	require.Contains(t, output.String(), "Deployment cell hc-provisioner: access request ar-auto is APPROVED")
}

func TestEnsureDeploymentCellAccessFallsBackWhenApproversCannotBeLoaded(t *testing.T) {
	var output bytes.Buffer
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", "reason", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-pending", Status: "PENDING", EnvironmentType: "GLOBAL"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-pending", Status: "APPROVED"}, nil
			},
			configuredApprovers: func(context.Context, string, string) ([]string, error) {
				return nil, assert.AnError
			},
			output:       &output,
			pollInterval: time.Millisecond,
		})

	require.NoError(t, err)
	require.Equal(t, "APPROVED", request.Status)
	require.Contains(t, output.String(), "Ask a configured approver")
}

func TestEnsureDeploymentCellAccessReportsDeniedDeletedRequest(t *testing.T) {
	notFound := &dataaccess.AccessRequestAPIError{StatusCode: http.StatusNotFound, Name: "not_found", Message: "access request not found"}
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", "reason", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-denied", Status: "PENDING"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				return nil, notFound
			},
			list: func(_ context.Context, _ string, options dataaccess.ListAccessRequestsOptions) (*model.AccessRequestList, error) {
				require.True(t, options.IncludeDeleted)
				return &model.AccessRequestList{Requests: []model.AccessRequest{{ID: "ar-denied", Status: "DENIED"}}}, nil
			},
			pollInterval: time.Millisecond,
		})

	require.Equal(t, "DENIED", request.Status)
	require.ErrorContains(t, err, "was denied")
	require.NotContains(t, err.Error(), "not_found")
}

func TestEnsureDeploymentCellAccessRequiresReasonForManualRequest(t *testing.T) {
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", "", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return nil, errors.New("bad_request\nDetail: manual access requires a nonblank reason")
			},
			describe: dataaccess.DescribeAccessRequest,
		})

	require.Nil(t, request)
	require.ErrorContains(t, err, "Specify a reason with --reason")
}

func TestEnsureDeploymentCellAccessManualWaitRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	request, err := ensureDeploymentCellAccess(
		ctx, "token", "hc-test", "cluster-admin", "reason", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-pending", Status: "PENDING"}, nil
			},
			describe: func(context.Context, string, string) (*model.AccessRequest, error) {
				t.Fatal("canceled request must not be polled")
				return nil, nil
			},
			pollInterval: time.Hour,
		})

	require.Equal(t, "ar-pending", request.ID)
	require.ErrorIs(t, err, context.Canceled)
}

func TestEnsureDeploymentCellAccessRejectsTerminalRequest(t *testing.T) {
	request, err := ensureDeploymentCellAccess(
		t.Context(), "token", "hc-test", "cluster-admin", "reason", deploymentCellAccessDependencies{
			create: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
				return &model.AccessRequest{ID: "ar-denied", Status: "DENIED"}, nil
			},
			describe: dataaccess.DescribeAccessRequest,
		})

	require.Equal(t, "ar-denied", request.ID)
	require.ErrorContains(t, err, "was denied")
}

func TestDeniedAccessRequestPresentation(t *testing.T) {
	request := &model.AccessRequest{ID: "ar-denied", Status: "DENIED"}
	err := errors.New("access request ar-denied was denied")

	require.True(t, IsDeniedAccessRequest(request, err))
	require.False(t, IsDeniedAccessRequest(request, nil))
	require.False(t, IsDeniedAccessRequest(&model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}, err))
	require.Equal(t, "Access request ar-denied was denied.", DeniedAccessRequestMessage(request))
}
