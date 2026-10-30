package deploymentcell

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestResolveKubeConfigAccessCreatesRequestBeforeFetchingKubeConfig(t *testing.T) {
	expected := &openapiclientfleet.KubeConfigHostClusterResult{Id: "hc-test"}
	createCalls := 0

	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			return expected, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			createCalls++
			return &model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}, nil
		},
	})

	require.NoError(t, err)
	require.Same(t, expected, result)
	require.Equal(t, "ar-approved", request.ID)
	require.Equal(t, 1, createCalls)
}

func TestResolveKubeConfigAccessWaitsAndContinuesAfterManualApproval(t *testing.T) {
	expected := &openapiclientfleet.KubeConfigHostClusterResult{Id: "hc-test"}
	waitCalls := 0
	approvalRequiredCalls := 0
	approvalGrantedCalls := 0
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			return expected, nil
		},
		createRequest: func(_ context.Context, _ string, input dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			require.Equal(t, "hc-test", input.DeploymentCellID)
			require.Equal(t, "cluster-admin", input.Role)
			require.Nil(t, input.Reason)
			return &model.AccessRequest{ID: "ar-pending", Status: "PENDING"}, nil
		},
		waitForApproval: func(_ context.Context, _ string, pending *model.AccessRequest) (*model.AccessRequest, error) {
			waitCalls++
			require.Equal(t, "ar-pending", pending.ID)
			return &model.AccessRequest{ID: pending.ID, Status: "APPROVED"}, nil
		},
		onApprovalRequired: func(pending *model.AccessRequest) {
			approvalRequiredCalls++
			require.Equal(t, "ar-pending", pending.ID)
		},
		onApprovalGranted: func(approved *model.AccessRequest) {
			approvalGrantedCalls++
			require.Equal(t, "APPROVED", approved.Status)
		},
	})

	require.NoError(t, err)
	require.Same(t, expected, result)
	require.Equal(t, "ar-pending", request.ID)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 1, waitCalls)
	require.Equal(t, 1, approvalRequiredCalls)
	require.Equal(t, 1, approvalGrantedCalls)
}

func TestResolveKubeConfigAccessRequiresReasonFlagForNewManualRequest(t *testing.T) {
	createCalls := 0
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "", "", kubeConfigAccessDependencies{
		getKubeConfig: accessRequestBlockedKubeConfig,
		createRequest: func(_ context.Context, _ string, input dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			createCalls++
			require.Equal(t, "cluster-reader", input.Role)
			require.Nil(t, input.Reason)
			return nil, errors.New("bad_request\nDetail: manual access requires a nonblank reason")
		},
	})

	require.ErrorContains(t, err, "requires --reason")
	require.Nil(t, result)
	require.Nil(t, request)
	require.Equal(t, 1, createCalls)
}

func TestResolveKubeConfigAccessUsesReasonFlag(t *testing.T) {
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-reader", "  Planned maintenance  ", kubeConfigAccessDependencies{
		getKubeConfig: accessRequestBlockedKubeConfig,
		createRequest: func(_ context.Context, _ string, input dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			require.NotNil(t, input.Reason)
			require.Equal(t, "Planned maintenance", *input.Reason)
			return &model.AccessRequest{ID: "ar-flag", Status: "PENDING"}, nil
		},
		waitForApproval: func(_ context.Context, _ string, pending *model.AccessRequest) (*model.AccessRequest, error) {
			return pending, nil
		},
	})

	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, "ar-flag", request.ID)
}

func TestResolveKubeConfigAccessRetriesUnchangedEndpointAfterApproval(t *testing.T) {
	getCalls := 0
	expected := &openapiclientfleet.KubeConfigHostClusterResult{Id: "hc-test"}
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			getCalls++
			return expected, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}, nil
		},
	})

	require.NoError(t, err)
	require.Same(t, expected, result)
	require.Equal(t, "ar-approved", request.ID)
	require.Equal(t, 1, getCalls)
}

func TestResolveKubeConfigAccessWaitsForAutomaticApprovalAndContinues(t *testing.T) {
	expected := &openapiclientfleet.KubeConfigHostClusterResult{Id: "hc-test"}
	waitCalls := 0
	getCalls := 0
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			getCalls++
			return expected, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: "ar-approving", Status: "APPROVING"}, nil
		},
		waitForApproval: func(_ context.Context, _ string, pending *model.AccessRequest) (*model.AccessRequest, error) {
			waitCalls++
			require.Equal(t, "ar-approving", pending.ID)
			return &model.AccessRequest{ID: pending.ID, Status: "APPROVED"}, nil
		},
	})

	require.NoError(t, err)
	require.Same(t, expected, result)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 1, waitCalls)
	require.Equal(t, 1, getCalls)
}

func TestResolveKubeConfigAccessWaitsWhenKubeConfigStillSeesApprovingRequest(t *testing.T) {
	expected := &openapiclientfleet.KubeConfigHostClusterResult{Id: "hc-test"}
	getCalls := 0
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			getCalls++
			if getCalls == 1 {
				return nil, errors.New("forbidden\nDetail: unauthorized: access request ar-approving is not approved")
			}
			return expected, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: "ar-approving", Status: "APPROVED"}, nil
		},
		kubeConfigWaitTimeout:  time.Second,
		kubeConfigPollInterval: time.Millisecond,
	})

	require.NoError(t, err)
	require.Same(t, expected, result)
	require.Equal(t, "ar-approving", request.ID)
	require.Equal(t, 2, getCalls)
}

func TestResolveKubeConfigAccessAllowsRerunWhenKubeConfigRemainsPending(t *testing.T) {
	getCalls := 0
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			getCalls++
			return nil, errors.New("forbidden\nDetail: unauthorized: access request ar-approving is not approved")
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: "ar-approving", Status: "APPROVED"}, nil
		},
		kubeConfigWaitTimeout:  10 * time.Millisecond,
		kubeConfigPollInterval: time.Millisecond,
	})

	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, "ar-approving", request.ID)
	require.Greater(t, getCalls, 1)
}

func TestResolveKubeConfigAccessAllowsRerunWhenAutomaticApprovalIsStillRunning(t *testing.T) {
	result, request, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "cluster-admin", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			t.Fatal("kubeconfig must not be fetched before approval")
			return nil, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: "ar-approving", Status: "APPROVING"}, nil
		},
		waitForApproval: func(_ context.Context, _ string, pending *model.AccessRequest) (*model.AccessRequest, error) {
			return pending, nil
		},
	})

	require.NoError(t, err)
	require.Nil(t, result)
	require.Equal(t, "ar-approving", request.ID)
	require.Equal(t, "APPROVING", request.Status)
}

func TestResolveKubeConfigAccessDoesNotMaskRequestCreationErrors(t *testing.T) {
	expectedErr := errors.New("forbidden\nDetail: access request role conflicts with the active request")
	_, _, err := resolveKubeConfigAccess(t.Context(), "token", "hc-test", "", "", kubeConfigAccessDependencies{
		getKubeConfig: func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
			t.Fatal("kubeconfig must not be fetched when request creation fails")
			return nil, nil
		},
		createRequest: func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error) {
			return nil, expectedErr
		},
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestWaitForAccessRequestApprovalPollsUntilApproved(t *testing.T) {
	describeCalls := 0
	request, err := waitForAccessRequestApproval(
		t.Context(),
		"token",
		&model.AccessRequest{ID: "ar-test", Status: "PENDING"},
		func(_ context.Context, token, requestID string) (*model.AccessRequest, error) {
			describeCalls++
			require.Equal(t, "token", token)
			require.Equal(t, "ar-test", requestID)
			if describeCalls == 1 {
				return &model.AccessRequest{ID: requestID, Status: "PENDING"}, nil
			}
			if describeCalls == 2 {
				return &model.AccessRequest{ID: requestID, Status: "APPROVING"}, nil
			}
			return &model.AccessRequest{ID: requestID, Status: "APPROVED"}, nil
		},
		time.Second,
		time.Millisecond,
	)

	require.NoError(t, err)
	require.Equal(t, "APPROVED", request.Status)
	require.Equal(t, 3, describeCalls)
}

func TestWaitForAccessRequestApprovalReturnsReusableRequestOnTimeout(t *testing.T) {
	request, err := waitForAccessRequestApproval(
		t.Context(),
		"token",
		&model.AccessRequest{ID: "ar-test", Status: "PENDING"},
		func(_ context.Context, _, requestID string) (*model.AccessRequest, error) {
			return &model.AccessRequest{ID: requestID, Status: "PENDING"}, nil
		},
		10*time.Millisecond,
		time.Millisecond,
	)

	require.NoError(t, err)
	require.Equal(t, "ar-test", request.ID)
	require.Equal(t, "PENDING", request.Status)
}

func TestWaitForAccessRequestApprovalWaitsWithoutTimeoutUntilCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	request, err := waitForAccessRequestApproval(
		ctx,
		"token",
		&model.AccessRequest{ID: "ar-test", Status: "PENDING"},
		func(_ context.Context, _, requestID string) (*model.AccessRequest, error) {
			cancel()
			return &model.AccessRequest{ID: requestID, Status: "PENDING"}, nil
		},
		0,
		time.Hour,
	)

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, "ar-test", request.ID)
	require.Equal(t, "PENDING", request.Status)
}

func accessRequestBlockedKubeConfig(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error) {
	return nil, errors.New("kubeconfig must not be fetched before access approval")
}
