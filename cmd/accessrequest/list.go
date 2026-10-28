package accessrequest

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "List access requests",
	Long:         "List access requests visible to the authenticated service-provider organization.",
	Args:         cobra.NoArgs,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringSlice("status", nil, "Filter by lifecycle status (comma-separated or repeated)")
	listCmd.Flags().Bool("include-deleted", false, "Include denied, revoked, and expired request history")
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
			cleanStatuses = append(cleanStatuses, status)
		}
	}
	includeDeleted, err := cmd.Flags().GetBool("include-deleted")
	if err != nil {
		return err
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
		Statuses:       cleanStatuses,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return fmt.Errorf("failed to list access requests: %w", err)
	}
	return utils.PrintTextTableJsonArrayOutput(output, result.Requests)
}
