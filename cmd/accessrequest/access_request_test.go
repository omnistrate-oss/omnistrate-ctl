package accessrequest

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAccessRequestCommands(t *testing.T) {
	require.Equal(t, "access-request [operation] [flags]", Cmd.Use)
	require.Equal(t, "Manage deployment-cell access requests", Cmd.Short)
	require.Contains(t, Cmd.Long, "Configure")

	expectedCommands := []string{"approve", "config", "deny", "describe", "list", "revoke"}
	require.Len(t, Cmd.Commands(), len(expectedCommands))
	for index, expected := range expectedCommands {
		require.Equal(t, expected, Cmd.Commands()[index].Name())
		command, _, err := Cmd.Find([]string{expected})
		require.NoError(t, err)
		require.Equal(t, expected, command.Name())
		require.NotNil(t, command.RunE)
		require.True(t, command.SilenceUsage)
	}
	require.ErrorContains(t, Cmd.ValidateArgs([]string{"create"}), "unknown command")
}

func TestAccessRequestCommandFlags(t *testing.T) {
	require.Equal(t, "[]", listCmd.Flags().Lookup("status").DefValue)
	require.Nil(t, listCmd.Flags().Lookup("include-deleted"))
	require.Equal(t, "0", approveCmd.Flags().Lookup("duration-minutes").DefValue)
	require.Equal(t, "", configGetCmd.Flags().Lookup("environment-type").DefValue)
	require.Equal(t, "", configSetCmd.Flags().Lookup("environment-type").DefValue)
	require.Equal(t, "", configSetCmd.Flags().Lookup("approval-mode").DefValue)
	require.Equal(t, "[]", configSetCmd.Flags().Lookup("approver-email").DefValue)
	require.Nil(t, configSetCmd.Flags().Lookup("approver-user-id"))
}

func TestAccessRequestCommandArguments(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		command *cobra.Command
		args    []string
		wantErr string
	}{
		{name: "describe accepts request", command: describeCmd, args: []string{"ar-12345678"}},
		{name: "list rejects arguments", command: listCmd, args: []string{"extra"}, wantErr: "unknown command"},
		{name: "approve accepts request", command: approveCmd, args: []string{"ar-12345678"}},
		{name: "approve requires request", command: approveCmd, wantErr: "accepts 1 arg(s)"},
		{name: "config get accepts no arguments", command: configGetCmd},
		{name: "config get rejects arguments", command: configGetCmd, args: []string{"extra"}, wantErr: "unknown command"},
		{name: "config set accepts no arguments", command: configSetCmd},
		{name: "config set rejects arguments", command: configSetCmd, args: []string{"extra"}, wantErr: "unknown command"},
		{name: "deny accepts request", command: denyCmd, args: []string{"ar-12345678"}},
		{name: "deny requires request", command: denyCmd, wantErr: "accepts 1 arg(s)"},
		{name: "revoke accepts request", command: revokeCmd, args: []string{"ar-12345678"}},
		{name: "revoke requires request", command: revokeCmd, wantErr: "accepts 1 arg(s)"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.command.ValidateArgs(testCase.args)
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestValidateAccessRequestConfiguration(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		mode          string
		approvers     []string
		wantMode      string
		wantApprovers []string
		wantErr       string
	}{
		{
			name:          "normalizes manual configuration",
			mode:          " manual ",
			approvers:     []string{" Approver@Example.com ", "approver@example.com", "second@example.com"},
			wantMode:      "MANUAL",
			wantApprovers: []string{"Approver@Example.com", "second@example.com"},
		},
		{
			name:          "accepts automatic prod configuration",
			mode:          "AUTO",
			wantMode:      "AUTO",
			wantApprovers: []string{},
		},
		{name: "rejects unsupported mode", mode: "PROMPT", wantErr: "AUTO or MANUAL"},
		{name: "requires manual approver", mode: "MANUAL", wantErr: "at least one"},
		{name: "rejects automatic approver", mode: "AUTO", approvers: []string{"approver@example.com"}, wantErr: "only valid with MANUAL"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mode, approvers, err := validateAccessRequestConfiguration(testCase.mode, testCase.approvers)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.wantMode, mode)
			require.Equal(t, testCase.wantApprovers, approvers)
		})
	}
}

func TestAccessRequestApprovalConfigurationTable(t *testing.T) {
	formatted := accessRequestApprovalConfigurationOutput{
		EnvironmentType: "GLOBAL",
		ApprovalMode:    "MANUAL",
		ApproverEmails:  []string{"first@example.com", "second@example.com"},
	}
	var output bytes.Buffer
	newAccessRequestApprovalConfigurationTable(formatted).PrintToWriter(&output)
	rendered := output.String()

	require.Less(t, strings.Index(rendered, "environmentType"), strings.Index(rendered, "approvalMode"))
	require.Less(t, strings.Index(rendered, "approvalMode"), strings.Index(rendered, "approverEmails"))
	require.Contains(t, rendered, "first@example.com, second@example.com")
	require.NotContains(t, rendered, "[first@example.com")
	require.NotContains(t, rendered, "defaulted")
}

func TestAutomaticAccessRequestApprovalConfigurationOmitsApprovers(t *testing.T) {
	formatted := accessRequestApprovalConfigurationOutput{
		EnvironmentType: "GLOBAL",
		ApprovalMode:    "AUTO",
	}
	var output bytes.Buffer
	newAccessRequestApprovalConfigurationTable(formatted).PrintToWriter(&output)
	rendered := output.String()

	require.Contains(t, rendered, "environmentType")
	require.Contains(t, rendered, "approvalMode")
	require.NotContains(t, rendered, "approverEmails")
	require.NotContains(t, rendered, "defaulted")

	encoded, err := json.Marshal(formatted)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "approverEmails")
}

func TestAccessRequestListTableOmitsEmptyLifecycleColumns(t *testing.T) {
	table, err := newAccessRequestTable([]model.AccessRequest{{
		ID:                   "ar-pending",
		Status:               "PENDING",
		DeploymentCellID:     "hc-test",
		Role:                 "cluster-admin",
		RequesterUserID:      "user-test",
		Reason:               "Testing access",
		EnvironmentType:      "GLOBAL",
		ServiceProviderOrgID: "org-test",
		CreatedAt:            "2026-11-02T23:27:46Z",
	}})
	require.NoError(t, err)

	var output bytes.Buffer
	table.PrintToWriter(&output)
	rendered := output.String()
	require.Less(t, strings.Index(rendered, "id"), strings.Index(rendered, "status"))
	require.Contains(t, rendered, "deploymentCellId")
	require.Contains(t, rendered, "createdAt")
	require.NotContains(t, rendered, "approvedBy")
	require.NotContains(t, rendered, "approvedAt")
	require.NotContains(t, rendered, "expiresAt")
	require.NotContains(t, rendered, "revokedBy")
	require.NotContains(t, rendered, "revokedAt")
}

func TestRequiredValue(t *testing.T) {
	value, err := requiredAccessRequestID("  ar-12345678  ")
	require.NoError(t, err)
	require.Equal(t, "ar-12345678", value)

	_, err = requiredAccessRequestID(" \t ")
	require.ErrorContains(t, err, "access request ID is required")
}

func TestOutputFormat(t *testing.T) {
	for _, testCase := range []struct {
		value   string
		wantErr bool
	}{
		{value: "text"},
		{value: "table"},
		{value: "json"},
		{value: "yaml", wantErr: true},
	} {
		command := &cobra.Command{}
		command.Flags().String(common.OutputFlag, testCase.value, "output")
		output, err := outputFormat(command)
		if testCase.wantErr {
			require.ErrorContains(t, err, "unsupported output format")
			continue
		}
		require.NoError(t, err)
		require.Equal(t, testCase.value, output)
	}
}
