package customer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "customer [operation] [flags]",
	Short:        "Manage customer portal users",
	Long:         "This command helps ISVs manage customer portal users.",
	Run:          runCustomer,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(verifyCmd)
	Cmd.AddCommand(suspendCmd)
	Cmd.AddCommand(unsuspendCmd)
	Cmd.AddCommand(deleteCmd)
}

func runCustomer(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}

func parseAttributes(values []string) (map[string]string, error) {
	attributes := make(map[string]string)
	for _, value := range values {
		for _, pair := range strings.Split(value, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			key, val, ok := strings.Cut(pair, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("invalid attribute %q. Attributes must use key=value format", pair)
			}
			attributes[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	}
	return attributes, nil
}

func formatUser(user fleet.AccessSideUser) model.CustomerUser {
	return model.CustomerUser{
		UserID:            stringValue(user.UserId),
		UserName:          stringValue(user.UserName),
		Email:             stringValue(user.Email),
		Status:            stringValue(user.Status),
		Enabled:           formatOptionalBool(user.Enabled),
		OrgID:             stringValue(user.OrgId),
		OrgName:           stringValue(user.OrgName),
		SubscriptionCount: formatOptionalInt64(user.SubscriptionCount),
		InstanceCount:     formatOptionalInt64(user.InstanceCount),
		CreatedAt:         stringValue(user.CreatedAt),
	}
}

func formatDescribeUser(user *fleet.FleetDescribeUserResult) model.CustomerUser {
	if user == nil {
		return model.CustomerUser{}
	}

	return model.CustomerUser{
		UserID:    stringValue(user.UserId),
		UserName:  stringValue(user.UserName),
		Email:     stringValue(user.Email),
		Status:    stringValue(user.Status),
		Enabled:   formatOptionalBool(user.Enabled),
		OrgID:     stringValue(user.OrgId),
		OrgName:   stringValue(user.OrgName),
		CreatedAt: stringValue(user.CreatedAt),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatOptionalBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}

func formatOptionalInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
