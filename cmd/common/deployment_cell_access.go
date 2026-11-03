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
	create              func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error)
	describe            func(context.Context, string, string) (*model.AccessRequest, error)
	list                func(context.Context, string, dataaccess.ListAccessRequestsOptions) (*model.AccessRequestList, error)
	configuredApprovers func(context.Context, string, string) ([]string, error)
	output              io.Writer
	pollInterval        time.Duration
	autoTimeout         time.Duration
}

var defaultDeploymentCellAccessDependencies = deploymentCellAccessDependencies{
	create:              dataaccess.CreateAccessRequest,
	describe:            dataaccess.DescribeAccessRequest,
	list:                dataaccess.ListAccessRequests,
	configuredApprovers: dataaccess.GetConfiguredAccessRequestApprovers,
	output:              os.Stderr,
	pollInterval:        deploymentCellAccessPollInterval,
	autoTimeout:         deploymentCellAccessApprovalTimeout,
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
			return nil, fmt.Errorf("JIT access requires manual approval for this organization. Specify a reason with --reason and try again")
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
				approvers := []string(nil)
				if dependencies.configuredApprovers != nil {
					approvers, _ = dependencies.configuredApprovers(
						waitCtx, token, request.EnvironmentType)
				}
				printDeploymentCellApprovalWait(dependencies.output, request.ID, approvers)
				approvalMessageShown = true
			}
		case "APPROVING":
			// Automatic approval is in progress; continue polling.
		default:
			return request, accessRequestStateError(request.ID, request.Status)
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return request, fmt.Errorf("access request %s did not reach approved status before timeout", request.ID)
			}
			return request, waitCtx.Err()
		case <-time.After(dependencies.pollInterval):
		}

		requestID := request.ID
		updatedRequest, err := dependencies.describe(waitCtx, token, requestID)
		if err != nil {
			if dataaccess.IsAccessRequestNotFound(err) {
				terminal := findDeletedAccessRequest(waitCtx, token, requestID, dependencies.list)
				if terminal != nil {
					return terminal, accessRequestStateError(terminal.ID, terminal.Status)
				}
				return request, fmt.Errorf("access request %s is no longer available", requestID)
			}
			return request, err
		}
		request = updatedRequest
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

func findDeletedAccessRequest(
	ctx context.Context,
	token string,
	requestID string,
	list func(context.Context, string, dataaccess.ListAccessRequestsOptions) (*model.AccessRequestList, error),
) *model.AccessRequest {
	if list == nil {
		return nil
	}
	requests, err := list(ctx, token, dataaccess.ListAccessRequestsOptions{
		Statuses:       []string{"DENIED", "REVOKED", "EXPIRED"},
		IncludeDeleted: true,
	})
	if err != nil || requests == nil {
		return nil
	}
	for index := range requests.Requests {
		if requests.Requests[index].ID == requestID {
			return &requests.Requests[index]
		}
	}
	return nil
}

func accessRequestStateError(requestID, status string) error {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DENIED":
		return fmt.Errorf("access request %s was denied", requestID)
	case "REVOKED":
		return fmt.Errorf("access request %s was revoked", requestID)
	case "EXPIRED":
		return fmt.Errorf("access request %s expired", requestID)
	default:
		return fmt.Errorf("access request %s is %s", requestID, status)
	}
}

// IsDeniedAccessRequest reports whether an access operation stopped because
// the known request reached DENIED state.
func IsDeniedAccessRequest(request *model.AccessRequest, err error) bool {
	return err != nil && request != nil && strings.EqualFold(request.Status, "DENIED")
}

// DeniedAccessRequestMessage returns the customer-facing denial message.
func DeniedAccessRequestMessage(request *model.AccessRequest) string {
	if request == nil {
		return "Access request was denied."
	}
	return fmt.Sprintf("Access request %s was denied.", request.ID)
}

func printDeploymentCellApprovalWait(output io.Writer, requestID string, approvers []string) {
	_, _ = fmt.Fprintf(output, "Access request %s is pending approval.\n", requestID)
	if len(approvers) > 0 {
		_, _ = fmt.Fprintf(output, "Configured approvers: %s\n", strings.Join(approvers, ", "))
	} else {
		_, _ = fmt.Fprintf(output, "Ask a configured approver to approve access request %s.\n", requestID)
	}
	_, _ = fmt.Fprintln(output, "Waiting for approval. This command will continue automatically once approved.")
}
