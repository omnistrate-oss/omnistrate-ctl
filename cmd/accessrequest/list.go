package accessrequest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "List access requests",
	Long:         "List access requests currently exposed by the backend to the authenticated service-provider organization. Revoked-request visibility is controlled by backend retention policy.",
	Args:         cobra.NoArgs,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringSlice("status", nil, "Filter backend-visible requests by lifecycle status (comma-separated or repeated)")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	statuses, err := cmd.Flags().GetStringSlice("status")
	if err != nil {
		return err
	}
	cleanStatuses := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if status = strings.TrimSpace(status); status != "" {
			cleanStatuses = append(cleanStatuses, strings.ToUpper(status))
		}
	}
	output, err := outputFormat(cmd)
	if err != nil {
		return err
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	result, err := dataaccess.ListAccessRequests(cmd.Context(), token, dataaccess.ListAccessRequestsOptions{
		Statuses: cleanStatuses,
	})
	if err != nil {
		return fmt.Errorf("failed to list access requests: %w", err)
	}
	return printAccessRequests(output, result.Requests)
}

var accessRequestTableColumns = []string{
	"id",
	"status",
	"deploymentCellId",
	"role",
	"requesterUserId",
	"reason",
	"environmentType",
	"serviceProviderOrgId",
	"createdAt",
	"approvedBy",
	"approvedAt",
	"expiresAt",
	"revokedBy",
	"revokedAt",
}

func printAccessRequests(output string, requests []model.AccessRequest) error {
	if output != "table" {
		return utils.PrintTextTableJsonArrayOutput(output, requests)
	}
	if len(requests) == 0 {
		return nil
	}
	table, err := newAccessRequestTable(requests)
	if err != nil {
		return err
	}
	table.Print()
	return nil
}

func newAccessRequestTable(requests []model.AccessRequest) (*utils.Table, error) {
	encodedRequests := make([]json.RawMessage, 0, len(requests))
	presentColumns := make(map[string]bool, len(accessRequestTableColumns))
	for _, request := range requests {
		encoded, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		encodedRequests = append(encodedRequests, encoded)
		var fields map[string]any
		if err = json.Unmarshal(encoded, &fields); err != nil {
			return nil, err
		}
		for field := range fields {
			presentColumns[field] = true
		}
	}

	columns := make([]string, 0, len(accessRequestTableColumns))
	for _, column := range accessRequestTableColumns {
		if presentColumns[column] {
			columns = append(columns, column)
		}
	}
	table := utils.NewTableWithColumnOrder(columns)
	for _, encoded := range encodedRequests {
		if err := table.AddRowFromJSON(encoded); err != nil {
			return nil, err
		}
	}
	return table, nil
}
