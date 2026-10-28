package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	accessRequestSmokeRole            = "cluster-reader"
	accessRequestServiceAccountPrefix = "system:serviceaccount:dataplane-agent:omnistrate-"
)

func verifyDeploymentCellAccessRequest(t *testing.T, ctx context.Context, instanceID string) {
	t.Helper()

	describedInstance := describeInstanceForAccessSmokeTest(t, ctx, instanceID)
	deploymentCellID := describedInstance.GetDeploymentCellID()
	require.NotEmpty(t, deploymentCellID, "running instance must expose its deployment cell ID")

	kubeconfigPath := t.TempDir() + "/kubeconfig"
	cmd.RootCmd.SetArgs([]string{
		"deployment-cell", "update-kubeconfig", deploymentCellID,
		"--role", accessRequestSmokeRole,
		"--kubeconfig", kubeconfigPath,
	})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	require.NoError(t, err)
	kubeconfig.Timeout = 30 * time.Second
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	require.NoError(t, err)

	identity, err := clientset.AuthenticationV1().SelfSubjectReviews().Create(
		ctx,
		&authenticationv1.SelfSubjectReview{},
		metav1.CreateOptions{},
	)
	require.NoError(t, err)
	requestIDFromIdentity := accessRequestIDFromServiceAccountUsername(identity.Status.UserInfo.Username)
	require.NotEmpty(t, requestIDFromIdentity, "unexpected Kubernetes identity %q", identity.Status.UserInfo.Username)

	_, err = clientset.CoreV1().Pods("dataplane-agent").List(ctx, metav1.ListOptions{Limit: 1})
	require.NoError(t, err, "cluster-reader access must allow listing dataplane-agent pods")

	request := findAccessRequestForSmokeTest(t, ctx, requestIDFromIdentity, deploymentCellID)
	require.Equal(t, "APPROVED", request.Status)
	require.Contains(t, []string{"cluster-reader", "cluster-admin"}, request.Role)
	require.Equal(t, deploymentCellID, request.DeploymentCellID)
	require.NotEmpty(t, request.EnvironmentType)
	require.NotEmpty(t, request.CreatedAt)

	describedRequest := describeAccessRequestForSmokeTest(t, ctx, request.ID)
	require.Equal(t, request.ID, describedRequest.ID)
	require.Equal(t, "APPROVED", describedRequest.Status)
	require.Equal(t, request.Role, describedRequest.Role)
	require.Equal(t, deploymentCellID, describedRequest.DeploymentCellID)

	requireAccessRequestAuditEvents(t, ctx, describedRequest)
}

func describeInstanceForAccessSmokeTest(
	t *testing.T,
	ctx context.Context,
	instanceID string,
) *openapiclientfleet.ResourceInstance {
	t.Helper()

	utils.LastPrintedString = ""
	cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID, "--output", "json"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
	require.NotEmpty(t, utils.LastPrintedString)

	var instance openapiclientfleet.ResourceInstance
	require.NoError(t, json.Unmarshal([]byte(utils.LastPrintedString), &instance))
	return &instance
}

func findAccessRequestForSmokeTest(
	t *testing.T,
	ctx context.Context,
	requestIDFromIdentity string,
	deploymentCellID string,
) *model.AccessRequest {
	t.Helper()

	utils.LastPrintedString = ""
	cmd.RootCmd.SetArgs([]string{"access-request", "list", "--output", "json"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
	require.NotEmpty(t, utils.LastPrintedString)

	var requests []model.AccessRequest
	require.NoError(t, json.Unmarshal([]byte(utils.LastPrintedString), &requests))
	for requestIndex := range requests {
		request := &requests[requestIndex]
		if strings.EqualFold(request.ID, requestIDFromIdentity) &&
			request.DeploymentCellID == deploymentCellID {
			return request
		}
	}
	require.FailNowf(t, "access request not found", "request %s for deployment cell %s was not listed", requestIDFromIdentity, deploymentCellID)
	return nil
}

func describeAccessRequestForSmokeTest(t *testing.T, ctx context.Context, requestID string) *model.AccessRequest {
	t.Helper()

	utils.LastPrintedString = ""
	cmd.RootCmd.SetArgs([]string{"access-request", "describe", requestID, "--output", "json"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
	require.NotEmpty(t, utils.LastPrintedString)

	var request model.AccessRequest
	require.NoError(t, json.Unmarshal([]byte(utils.LastPrintedString), &request))
	return &request
}

func requireAccessRequestAuditEvents(t *testing.T, ctx context.Context, request *model.AccessRequest) {
	t.Helper()

	createdAt, err := time.Parse(time.RFC3339Nano, request.CreatedAt)
	require.NoError(t, err)
	startDate := createdAt.Add(-time.Minute).UTC().Format(time.RFC3339)

	var (
		lastErr       error
		lastEvents    []openapiclientfleet.FleetAuditEvent
		foundCreated  bool
		foundApproved bool
	)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		utils.LastPrintedString = ""
		cmd.RootCmd.SetArgs([]string{
			"audit", "list",
			"--event-source-types", "AccessRequestEvent",
			"--environment-type", request.EnvironmentType,
			"--start-date", startDate,
			"--page-size", "100",
			"--output", "json",
		})
		lastErr = cmd.RootCmd.ExecuteContext(ctx)
		if lastErr == nil {
			var results []openapiclientfleet.FleetAuditEventsResult
			lastErr = json.Unmarshal([]byte(utils.LastPrintedString), &results)
			if lastErr == nil {
				lastEvents = nil
				for resultIndex := range results {
					lastEvents = append(lastEvents, results[resultIndex].Events...)
				}
				foundCreated, foundApproved = accessRequestAuditEventsPresent(lastEvents, request.ID)
				if foundCreated && foundApproved {
					return
				}
			}
		}
		time.Sleep(2 * time.Second)
	}

	require.NoError(t, lastErr)
	require.Truef(t, foundCreated, "created audit event for access request %s was not returned; events: %s", request.ID, auditEventSummary(lastEvents))
	require.Truef(t, foundApproved, "approved audit event for access request %s was not returned; events: %s", request.ID, auditEventSummary(lastEvents))
}

func accessRequestIDFromServiceAccountUsername(username string) string {
	if !strings.HasPrefix(strings.ToLower(username), accessRequestServiceAccountPrefix) {
		return ""
	}
	return username[len(accessRequestServiceAccountPrefix):]
}

func accessRequestAuditEventsPresent(events []openapiclientfleet.FleetAuditEvent, requestID string) (created, approved bool) {
	for eventIndex := range events {
		event := &events[eventIndex]
		if event.EventSource == nil || *event.EventSource != "AccessRequestEvent" || !strings.Contains(event.Message, requestID) {
			continue
		}
		switch event.ResourceName {
		case "Access request created":
			created = true
		case "Access request approved":
			approved = true
		}
	}
	return
}

func auditEventSummary(events []openapiclientfleet.FleetAuditEvent) string {
	summaries := make([]string, 0, len(events))
	for eventIndex := range events {
		event := &events[eventIndex]
		summaries = append(summaries, fmt.Sprintf("%s: %s", event.ResourceName, event.Message))
	}
	return strings.Join(summaries, "; ")
}
