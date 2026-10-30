package common

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

const (
	deploymentCellAccessApprovalTimeout = 5 * time.Minute
	deploymentCellAccessPollInterval    = 2 * time.Second
)

type deploymentCellAccessDependencies struct {
	create       func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error)
	describe     func(context.Context, string, string) (*model.AccessRequest, error)
	output       io.Writer
	pollInterval time.Duration
	autoTimeout  time.Duration
}

var defaultDeploymentCellAccessDependencies = deploymentCellAccessDependencies{
	create:       dataaccess.CreateAccessRequest,
	describe:     dataaccess.DescribeAccessRequest,
	output:       os.Stderr,
	pollInterval: deploymentCellAccessPollInterval,
	autoTimeout:  deploymentCellAccessApprovalTimeout,
}

// EnsureDeploymentCellAccess creates or reuses an access request and waits
// until access is approved. Manual requests wait until the caller cancels.
func EnsureDeploymentCellAccess(
	ctx context.Context,
	token string,
	deploymentCellID string,
	role string,
	reason string,
) (*model.AccessRequest, error) {
	return ensureDeploymentCellAccess(
		ctx, token, deploymentCellID, role, reason, defaultDeploymentCellAccessDependencies)
}

func ensureDeploymentCellAccess(
	ctx context.Context,
	token string,
	deploymentCellID string,
	role string,
	reason string,
	dependencies deploymentCellAccessDependencies,
) (*model.AccessRequest, error) {
	if dependencies.create == nil || dependencies.describe == nil {
		return nil, fmt.Errorf("deployment-cell access request operations are not configured")
	}
	if dependencies.output == nil {
		dependencies.output = io.Discard
	}
	if dependencies.pollInterval <= 0 {
		dependencies.pollInterval = deploymentCellAccessPollInterval
	}
	if dependencies.autoTimeout <= 0 {
		dependencies.autoTimeout = deploymentCellAccessApprovalTimeout
	}

	role = strings.TrimSpace(role)
	if role == "" {
		role = "cluster-reader"
	}
	reason = strings.TrimSpace(reason)
	request, err := dependencies.create(ctx, token, dataaccess.CreateAccessRequestRequest{
		DeploymentCellID: deploymentCellID,
		Role:             role,
		Reason:           optionalAccessReason(reason),
	})
	if err != nil {
		if reason == "" && dataaccess.IsManualAccessReasonRequiredError(err) {
			return nil, fmt.Errorf("manual deployment-cell access requires --reason")
		}
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("access request API returned no request")
	}

	waitCtx := ctx
	cancel := func() {}
	if strings.EqualFold(request.Status, "APPROVING") {
		waitCtx, cancel = context.WithTimeout(ctx, dependencies.autoTimeout)
	}
	defer cancel()

	approvalMessageShown := false
	for {
		switch strings.ToUpper(request.Status) {
		case "APPROVED":
			return request, nil
		case "PENDING":
			if !approvalMessageShown {
				printDeploymentCellApprovalWait(dependencies.output, request.ID)
				approvalMessageShown = true
			}
		case "APPROVING":
			// Automatic approval is in progress; continue polling.
		default:
			return request, fmt.Errorf("access request %s is %s", request.ID, request.Status)
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return request, fmt.Errorf("access request %s did not reach approved status before timeout", request.ID)
			}
			return request, waitCtx.Err()
		case <-time.After(dependencies.pollInterval):
		}

		request, err = dependencies.describe(waitCtx, token, request.ID)
		if err != nil {
			return request, err
		}
		if request == nil {
			return nil, fmt.Errorf("access request API returned no request while waiting for approval")
		}
	}
}

func optionalAccessReason(reason string) *string {
	if reason == "" {
		return nil
	}
	return &reason
}

func printDeploymentCellApprovalWait(output io.Writer, requestID string) {
	_, _ = fmt.Fprintf(output, "Access request %s is pending approval.\n", requestID)
	_, _ = fmt.Fprintf(output, "Ask a configured approver to approve access request %s.\n", requestID)
	_, _ = fmt.Fprintln(output, "Waiting for approval. This command will continue automatically once approved.")
}
