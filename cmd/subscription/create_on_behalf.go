package subscription

import (
	"context"
	"encoding/json"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var createOnBehalfCmd = &cobra.Command{
	Use:   "create-on-behalf",
	Short: "Create subscription on behalf of customer",
	Long:  "Create a subscription on behalf of a customer for a service environment.",
	RunE:  runCreateOnBehalf,
}

func init() {
	createOnBehalfCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	createOnBehalfCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")
	createOnBehalfCmd.Flags().String("product-tier-id", "", "Product tier ID (required)")
	createOnBehalfCmd.Flags().String("customer-user-id", "", "Customer user ID (required)")
	createOnBehalfCmd.Flags().Bool("allow-creates-without-payment", false, "Allow creation without payment configured")
	createOnBehalfCmd.Flags().String("billing-provider", "", "Billing provider")
	createOnBehalfCmd.Flags().Bool("custom-price", false, "Whether to use custom price")
	createOnBehalfCmd.Flags().String("custom-price-per-unit", "", "Custom price per unit (JSON object)")
	createOnBehalfCmd.Flags().String("external-payer-id", "", "External payer ID")
	createOnBehalfCmd.Flags().Int64("max-instances", 0, "Maximum number of instances")
	createOnBehalfCmd.Flags().String("price-effective-date", "", "Price effective date")

	_ = createOnBehalfCmd.MarkFlagRequired("service-id")
	_ = createOnBehalfCmd.MarkFlagRequired("environment-id")
	_ = createOnBehalfCmd.MarkFlagRequired("product-tier-id")
	_ = createOnBehalfCmd.MarkFlagRequired("customer-user-id")
}

func runCreateOnBehalf(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")
	customerUserID, _ := cmd.Flags().GetString("customer-user-id")
	allowWithoutPayment, _ := cmd.Flags().GetBool("allow-creates-without-payment")
	billingProvider, _ := cmd.Flags().GetString("billing-provider")
	customPrice, _ := cmd.Flags().GetBool("custom-price")
	customPricePerUnitStr, _ := cmd.Flags().GetString("custom-price-per-unit")
	externalPayerID, _ := cmd.Flags().GetString("external-payer-id")
	maxInstances, _ := cmd.Flags().GetInt64("max-instances")
	priceEffectiveDate, _ := cmd.Flags().GetString("price-effective-date")

	// Parse custom price per unit if provided
	var customPricePerUnit map[string]interface{}
	if customPricePerUnitStr != "" {
		if err := json.Unmarshal([]byte(customPricePerUnitStr), &customPricePerUnit); err != nil {
			return err
		}
	}

	opts := &dataaccess.CreateSubscriptionOnBehalfOptions{
		ProductTierID:            productTierID,
		OnBehalfOfCustomerUserID: customerUserID,
		BillingProvider:          billingProvider,
		ExternalPayerID:          externalPayerID,
		PriceEffectiveDate:       priceEffectiveDate,
		CustomPricePerUnit:       customPricePerUnit,
	}

	if cmd.Flags().Changed("allow-creates-without-payment") {
		opts.AllowCreatesWhenPaymentNotConfigured = &allowWithoutPayment
	}
	if cmd.Flags().Changed("custom-price") {
		opts.CustomPrice = &customPrice
	}
	if cmd.Flags().Changed("max-instances") {
		opts.MaxNumberOfInstances = &maxInstances
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.CreateSubscriptionOnBehalf(ctx, token, serviceID, environmentID, opts)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}