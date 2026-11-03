package accessrequest

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

func requiredAccessRequestID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("access request ID is required")
	}
	return value, nil
}

func outputFormat(cmd *cobra.Command) (string, error) {
	output, err := cmd.Flags().GetString(common.OutputFlag)
	if err != nil {
		return "", err
	}
	switch output {
	case "text", "table", "json":
		return output, nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", output)
	}
}

func printAccessRequest(output string, request *model.AccessRequest) error {
	return utils.PrintTextTableJsonOutput(output, request)
}
