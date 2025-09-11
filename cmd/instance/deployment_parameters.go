package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var (
	deploymentParamsOutputFlag string
)

var deploymentParametersCmd = &cobra.Command{
	Use:   "deployment-parameters --service=[service] --plan=[plan] --version=[version] --resource=[resource]",
	Short: "List API parameters configurable for instance deployment",
	Long:  `This command retrieves and displays the configurable API parameters from the service offerings API that can be used during instance deployment.`,
	Example: `  omnistrate-ctl instance deployment-parameters --service=mysql --plan=mysql --version=latest --resource=mySQL
  omnistrate-ctl instance deployment-parameters --service=mysql --plan=mysql --version=latest --resource=mySQL --output=json`,
	RunE:         runDeploymentParameters,
	SilenceUsage: true,
}

func init() {
	deploymentParametersCmd.Flags().String("service", "", "Service name")
	deploymentParametersCmd.Flags().String("plan", "", "Service plan name")
	deploymentParametersCmd.Flags().String("version", "preferred", "Service plan version (latest|preferred|1.0 etc.)")
	deploymentParametersCmd.Flags().String("resource", "", "Resource name")
	deploymentParametersCmd.Flags().StringVarP(&deploymentParamsOutputFlag, "output", "o", "table", "Output format (table|json)")

	// Mark required flags
	if err := deploymentParametersCmd.MarkFlagRequired("service"); err != nil {
		utils.PrintError(err)
	}
	if err := deploymentParametersCmd.MarkFlagRequired("plan"); err != nil {
		utils.PrintError(err)
	}
	if err := deploymentParametersCmd.MarkFlagRequired("resource"); err != nil {
		utils.PrintError(err)
	}
}

type ParameterInfo struct {
	Key          string   `json:"key"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Modifiable   bool     `json:"modifiable"`
	IsList       bool     `json:"isList"`
	DefaultValue *string  `json:"defaultValue,omitempty"`
	Options      []string `json:"options,omitempty"`
	Regex        *string  `json:"regex,omitempty"`
	Custom       bool     `json:"custom"`
	API          string   `json:"api"`
}

type DeploymentParametersOutput struct {
	ServiceName  string          `json:"serviceName"`
	PlanName     string          `json:"planName"`
	Version      string          `json:"version"`
	ResourceName string          `json:"resourceName"`
	Parameters   []ParameterInfo `json:"parameters"`
}

func runDeploymentParameters(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Get flag values
	serviceName, err := cmd.Flags().GetString("service")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	planName, err := cmd.Flags().GetString("plan")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	version, err := cmd.Flags().GetString("version")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	resourceName, err := cmd.Flags().GetString("resource")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()

	// Use SearchInventory to find the resource and get service/plan info
	serviceID, resourceID, productTierID, productTierVersion, err := getServiceAndPlanInfo(ctx, token, serviceName, planName, resourceName, version)
	if err != nil {
		return fmt.Errorf("failed to get service and plan info: %w", err)
	}

	// If we don't have a product tier version, we need to get it from the service offering
	if productTierVersion == "" {
		serviceOfferingResult, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, productTierID, "")
		if err != nil {
			return fmt.Errorf("failed to describe service offering to get version: %w", err)
		}

		serviceOffering := serviceOfferingResult.GetConsumptionDescribeServiceOfferingResult()
		for _, offering := range serviceOffering.GetOfferings() {
			if offering.GetProductTierID() == productTierID {
				productTierVersion = offering.GetProductTierVersion()
				break
			}
		}

		if productTierVersion == "" {
			return fmt.Errorf("could not determine product tier version for plan %s", planName)
		}
	}

	// Get input parameters for the resource using the dedicated input parameter API
	parametersResult, err := dataaccess.ListInputParameters(ctx, token, serviceID, resourceID, productTierID, productTierVersion)
	if err != nil {
		return fmt.Errorf("failed to list input parameters: %w", err)
	}

	// Extract parameters from the result
	var parameters []ParameterInfo
	for _, param := range parametersResult.GetInputParameters() {
		var defaultValue *string
		if val, ok := param.GetDefaultValueOk(); ok && val != nil {
			defaultValue = val
		}

		var regex *string
		if val, ok := param.GetRegexOk(); ok && val != nil {
			regex = val
		}

		var options []string
		if param.GetOptions() != nil {
			options = param.GetOptions()
		}

		paramInfo := ParameterInfo{
			Key:          param.GetKey(),
			DisplayName:  param.GetName(),
			Description:  param.GetDescription(),
			Type:         param.GetType(),
			Required:     param.GetRequired(),
			Modifiable:   param.GetModifiable(),
			IsList:       param.GetIsList(),
			DefaultValue: defaultValue,
			Options:      options,
			Regex:        regex,
			Custom:       false,    // Not available in v1 API
			API:          "create", // Default API for deployment parameters
		}
		parameters = append(parameters, paramInfo)
	}

	// Sort parameters by key for consistent output
	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Key < parameters[j].Key
	})

	// Prepare output
	output := DeploymentParametersOutput{
		ServiceName:  serviceName,
		PlanName:     planName,
		Version:      version,
		ResourceName: resourceName,
		Parameters:   parameters,
	}

	// Handle output format
	switch deploymentParamsOutputFlag {
	case "json":
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	case "table":
		printParametersTable(output)
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats are table and json", deploymentParamsOutputFlag)
	}

	return nil
}

// getServiceAndPlanInfo uses SearchInventory to find service, plan, and resource info
func getServiceAndPlanInfo(ctx context.Context, token, serviceName, planName, resourceName, version string) (serviceID, resourceID, productTierID, productTierVersion string, err error) {
	// Search for the specific resource
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resource:%s", resourceName))
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to search inventory: %w", err)
	}

	// Find matching resource
	for _, res := range searchRes.ResourceResults {
		if res.Id == "" {
			continue
		}
		if strings.EqualFold(res.Name, resourceName) &&
			strings.EqualFold(res.ServiceName, serviceName) &&
			strings.EqualFold(res.ProductTierName, planName) {

			serviceID = res.ServiceId
			resourceID = res.Id
			productTierID = res.ProductTierId

			// Handle version
			if version != "preferred" && version != "latest" {
				productTierVersion = version
			}
			// For preferred/latest, leave productTierVersion empty

			return serviceID, resourceID, productTierID, productTierVersion, nil
		}
	}

	return "", "", "", "", fmt.Errorf("resource '%s' not found for service '%s' and plan '%s'", resourceName, serviceName, planName)
}

// printParametersTable prints the parameters in a table format
func printParametersTable(output DeploymentParametersOutput) {
	fmt.Printf("Deployment Parameters for %s/%s (version: %s, resource: %s)\n\n",
		output.ServiceName, output.PlanName, output.Version, output.ResourceName)

	if len(output.Parameters) == 0 {
		fmt.Println("No configurable parameters found.")
		return
	}

	// Print header
	fmt.Printf("%-25s %-20s %-12s %-12s %-8s %s\n",
		"PARAMETER KEY", "DISPLAY NAME", "TYPE", "REQUIRED", "API", "DESCRIPTION")
	fmt.Printf("%-25s %-20s %-12s %-12s %-8s %s\n",
		strings.Repeat("-", 25), strings.Repeat("-", 20), strings.Repeat("-", 12),
		strings.Repeat("-", 12), strings.Repeat("-", 8), strings.Repeat("-", 20))

	// Print parameters
	for _, param := range output.Parameters {
		required := "false"
		if param.Required {
			required = "true"
		}

		displayName := param.DisplayName
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}

		description := param.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Printf("%-25s %-20s %-12s %-12s %-8s %s\n",
			param.Key, displayName, param.Type, required, param.API, description)
	}

	fmt.Printf("\nTotal parameters: %d\n", len(output.Parameters))
	fmt.Println("\nUse --output=json for detailed parameter information including default values, options, and validation rules.")
}
