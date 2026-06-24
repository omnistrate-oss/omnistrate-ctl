package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

var (
	deploymentParamsOutputFlag string
)

var deploymentParametersCmd = &cobra.Command{
	Use:   "deployment-parameters --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource]",
	Short: "List API parameters configurable for instance deployment",
	Long:  `This command retrieves and displays the configurable API parameters from the service offerings API that can be used during instance deployment.`,
	Example: `  omnistrate-ctl instance deployment-parameters --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL
  omnistrate-ctl instance deployment-parameters --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --output=json`,
	RunE:         runDeploymentParameters,
	SilenceUsage: true,
}

func init() {
	deploymentParametersCmd.Flags().String("service", "", "Service name")
	deploymentParametersCmd.Flags().String("plan", "", "Service plan name")
	deploymentParametersCmd.Flags().String("version", "preferred", "Service plan version (latest|preferred|1.0 etc.)")
	deploymentParametersCmd.Flags().String("resource", "", "Resource name")
	deploymentParametersCmd.Flags().StringVarP(&deploymentParamsOutputFlag, "output", "o", "table", "Output format (table|json)")
	deploymentParametersCmd.Flags().String("environment", "", "Environment name or ID")

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
	if err := deploymentParametersCmd.MarkFlagRequired("environment"); err != nil {
		utils.PrintError(err)
	}

	templateCmd.Flags().String("service", "", "Service name")
	templateCmd.Flags().String("plan", "", "Service plan name")
	templateCmd.Flags().String("version", "preferred", "Service plan version (latest|preferred|1.0 etc.)")
	templateCmd.Flags().String("resource", "", "Resource name")
	templateCmd.Flags().String("environment", "", "Environment name or ID")

	for _, required := range []string{"service", "plan", "resource", "environment"} {
		if err := templateCmd.MarkFlagRequired(required); err != nil {
			utils.PrintError(err)
		}
	}

	deploymentParametersCmd.AddCommand(templateCmd)
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

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()

	output, err := fetchDeploymentParameters(ctx, token, serviceName, planName, version, resourceName, environment)
	if err != nil {
		return err
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

// fetchDeploymentParameters resolves the resource and returns its configurable
// deployment parameters. Shared by the table/json output path and the template path.
func fetchDeploymentParameters(ctx context.Context, token, serviceName, planName, version, resourceName, environment string) (DeploymentParametersOutput, error) {
	serviceID, resourceID, productTierID, productTierVersion, err := getServiceAndPlanInfo(ctx, token, serviceName, planName, resourceName, version, environment)
	if err != nil {
		return DeploymentParametersOutput{}, fmt.Errorf("failed to get service and plan info: %w", err)
	}

	// If we don't have a product tier version, get it from the service offering
	if productTierVersion == "" {
		serviceOfferingResult, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, productTierID, "")
		if err != nil {
			return DeploymentParametersOutput{}, fmt.Errorf("failed to describe service offering to get version: %w", err)
		}

		serviceOffering := serviceOfferingResult.GetConsumptionDescribeServiceOfferingResult()
		for _, offering := range serviceOffering.GetOfferings() {
			if offering.GetProductTierID() == productTierID {
				productTierVersion = offering.GetProductTierVersion()
				break
			}
		}

		if productTierVersion == "" {
			return DeploymentParametersOutput{}, fmt.Errorf("could not determine product tier version for plan %s", planName)
		}
	}

	parametersResult, err := dataaccess.ListInputParameters(ctx, token, serviceID, resourceID, productTierID, productTierVersion)
	if err != nil {
		return DeploymentParametersOutput{}, fmt.Errorf("failed to list input parameters: %w", err)
	}

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

		parameters = append(parameters, ParameterInfo{
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
			Custom:       false,
			API:          "create",
		})
	}

	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Key < parameters[j].Key
	})

	return DeploymentParametersOutput{
		ServiceName:  serviceName,
		PlanName:     planName,
		Version:      version,
		ResourceName: resourceName,
		Parameters:   parameters,
	}, nil
}

var templateCmd = &cobra.Command{
	Use:   "template --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource]",
	Short: "Generate a JSON parameter template for instance create",
	Long:  `This command generates a JSON parameter template for the given resource that can be filled in and passed to 'instance create --param-file'. Parameters with defaults are pre-populated; others get typed placeholders.`,
	Example: `  omnistrate-ctl instance deployment-parameters template --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL > params.json
  omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --resource=mySQL --cloud-provider=aws --region=us-east-2 --param-file params.json`,
	RunE:         runDeploymentParametersTemplate,
	SilenceUsage: true,
}

func runDeploymentParametersTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

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
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	output, err := fetchDeploymentParameters(context.Background(), token, serviceName, planName, version, resourceName, environment)
	if err != nil {
		return err
	}

	template := buildParamTemplate(output.Parameters)
	jsonData, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template to JSON: %w", err)
	}
	fmt.Println(string(jsonData))

	return nil
}

// findMatchingResource selects the resource record matching the service, plan,
// resource name, and environment (name or ID). Comparisons are case-insensitive.
func findMatchingResource(
	records []openapiclientfleet.ResourceSearchRecord,
	serviceName, planName, resourceName, environment string,
) (*openapiclientfleet.ResourceSearchRecord, error) {
	for i := range records {
		res := records[i]
		if res.Id == "" {
			continue
		}
		if !strings.EqualFold(res.Name, resourceName) ||
			!strings.EqualFold(res.ServiceName, serviceName) ||
			!strings.EqualFold(res.ProductTierName, planName) {
			continue
		}
		if !matchesIDOrName(res.ServiceEnvironmentId, res.ServiceEnvironmentName, environment) {
			continue
		}
		return &records[i], nil
	}
	return nil, fmt.Errorf(
		"resource '%s' not found for service '%s', plan '%s', and environment '%s'",
		resourceName, serviceName, planName, environment)
}

// getServiceAndPlanInfo uses SearchInventory to find service, plan, and resource info
func getServiceAndPlanInfo(ctx context.Context, token, serviceName, planName, resourceName, version, environment string) (serviceID, resourceID, productTierID, productTierVersion string, err error) {
	// Search for the specific resource
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resource:%s", resourceName))
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to search inventory: %w", err)
	}

	match, err := findMatchingResource(searchRes.ResourceResults, serviceName, planName, resourceName, environment)
	if err != nil {
		return "", "", "", "", err
	}

	serviceID = match.ServiceId
	resourceID = match.Id
	productTierID = match.ProductTierId

	// Handle version: only pin a concrete version, leave empty for preferred/latest
	if version != "preferred" && version != "latest" {
		productTierVersion = version
	}

	return serviceID, resourceID, productTierID, productTierVersion, nil
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

// placeholderForType returns a typed zero value for a parameter with no default.
func placeholderForType(valueType string) any {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "boolean":
		return false
	case "int", "int32", "int64", "integer":
		return 0
	case "float", "float32", "float64", "double", "number":
		return 0
	case "object", "json", "map":
		return map[string]any{}
	default:
		return ""
	}
}

// coerceParamValue converts a string default into a typed JSON value based on the
// parameter type. On any parse failure it returns the raw string unchanged.
func coerceParamValue(value, valueType string, isList bool) any {
	value = strings.TrimSpace(value)
	if isList {
		if strings.HasPrefix(value, "[") {
			var parsed any
			if err := json.Unmarshal([]byte(value), &parsed); err == nil {
				return parsed
			}
			return value
		}
		parts := strings.Split(value, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values
	}

	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "boolean":
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	case "int", "int32", "int64", "integer":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	case "float", "float32", "float64", "double", "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "object", "json", "map":
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}
	return value
}

// templateValueForParam returns the JSON template value for a single parameter:
// its (typed) default when present, otherwise a typed placeholder.
func templateValueForParam(p ParameterInfo) any {
	if p.DefaultValue != nil && strings.TrimSpace(*p.DefaultValue) != "" {
		return coerceParamValue(*p.DefaultValue, p.Type, p.IsList)
	}
	if p.IsList {
		return []any{}
	}
	return placeholderForType(p.Type)
}

// buildParamTemplate builds a key -> value map suitable for `instance create --param-file`.
func buildParamTemplate(params []ParameterInfo) map[string]any {
	template := make(map[string]any, len(params))
	for _, p := range params {
		template[p.Key] = templateValueForParam(p)
	}
	return template
}
