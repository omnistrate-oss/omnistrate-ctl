package accessrequest

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:          "describe [access-request-id]",
	Short:        "Describe an access request",
	Long:         "Describe an access request visible to the authenticated service-provider user.",
	Args:         cobra.ExactArgs(1),
	RunE:         runDescribe,
	SilenceUsage: true,
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accessRequestID, err := requiredAccessRequestID(args[0])
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
	result, err := dataaccess.DescribeAccessRequest(cmd.Context(), token, accessRequestID)
	if err != nil {
		return fmt.Errorf("failed to describe access request: %w", err)
	}
	return printAccessRequest(output, result)
}
