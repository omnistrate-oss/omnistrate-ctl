package common

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/require"
)

func TestEnsureDeploymentCellAccessReturnsApprovedRequest(t *testing.T) {
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
		})

	require.NoError(t, err)
	require.Equal(t, "ar-approved", request.ID)
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
			output:       &output,
			pollInterval: time.Millisecond,
			autoTimeout:  time.Second,
		})

	require.NoError(t, err)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 2, describeCalls)
	require.Contains(t, output.String(), "Access request ar-pending is pending approval.")
	require.Contains(t, output.String(), "Ask a configured approver")
	require.NotContains(t, output.String(), "http")
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
	require.ErrorContains(t, err, "requires --reason")
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
	require.ErrorContains(t, err, "is DENIED")
}
