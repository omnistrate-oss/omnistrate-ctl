package compose

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// UpdateCmd represents the update command under compose
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing Docker Compose spec with new parameters",
	Long: `Update an existing Docker Compose spec with Omnistrate extensions based on provided parameters.
This command reads an existing compose file and updates it with new configuration while preserving existing settings.

The command supports updating:
- Service configurations (image, ports, environment variables, etc.)
- Omnistrate-specific extensions (compute, capabilities, API parameters)
- Service plan name and integrations
- Adding new services or modifying existing ones

Examples:

Update service image and add environment variables:
  omnistrate-ctl compose update \
    --input-file docker-compose.yml \
    --service-name web \
    --image nginx:1.21 \
    --environment "NODE_ENV=production" \
    --environment "DEBUG=false"

Update multiple services with JSON configuration:
  omnistrate-ctl compose update \
    --input-file docker-compose.yml \
    --services '[
      {
        "name": "web",
        "image": "nginx:1.21",
        "replicaCount": 3
      },
      {
        "name": "api",
        "environment": ["NODE_ENV=production", "LOG_LEVEL=info"]
      }
    ]'

Update service plan name and add integrations:
  omnistrate-ctl compose update \
    --input-file docker-compose.yml \
    --service-plan-name "Updated Service Plan" \
    --integrations omnistrateLogging \
    --integrations omnistrateMetrics`,
	RunE: runUpdateCompose,
}

func init() {
	Cmd.AddCommand(UpdateCmd)
	
	// Input/Output flags
	UpdateCmd.Flags().StringP("input-file", "", "", "Input compose file path (required)")
	UpdateCmd.MarkFlagRequired("input-file")
	UpdateCmd.Flags().StringP("output-file", "f", "", "Output file path (default: overwrite input file)")
	
	// Service Plan flags
	UpdateCmd.Flags().StringP("service-plan-name", "n", "", "Update the service plan name")
	
	// Multiple services configuration
	UpdateCmd.Flags().StringP("services", "", "", "Services configuration updates as JSON string")
	
	// Single service update flags
	UpdateCmd.Flags().StringP("service-name", "s", "", "Name of the service to update (use with other single service flags)")
	UpdateCmd.Flags().StringP("image", "i", "", "Update Docker image for the service")
	UpdateCmd.Flags().StringArray("ports", nil, "Update port mappings (replaces existing ports)")
	UpdateCmd.Flags().StringArray("environment", nil, "Update environment variables (replaces existing environment)")
	UpdateCmd.Flags().StringArray("volumes", nil, "Update volume mounts (replaces existing volumes)")
	UpdateCmd.Flags().StringArray("add-ports", nil, "Add port mappings to existing ports")
	UpdateCmd.Flags().StringArray("add-environment", nil, "Add environment variables to existing environment")
	UpdateCmd.Flags().StringArray("add-volumes", nil, "Add volume mounts to existing volumes")
	UpdateCmd.Flags().IntP("root-volume-size", "", 0, "Update root volume size in GB (0 = no change)")
	UpdateCmd.Flags().IntP("replica-count", "", 0, "Update number of replicas (0 = no change)")
	UpdateCmd.Flags().StringP("replica-count-api-param", "", "", "Update API parameter name for replica count")
	UpdateCmd.Flags().BoolP("enable-multi-zone", "", false, "Enable multi-zone deployment")
	UpdateCmd.Flags().BoolP("disable-multi-zone", "", false, "Disable multi-zone deployment")
	UpdateCmd.Flags().BoolP("enable-endpoint-per-replica", "", false, "Enable endpoint per replica")
	UpdateCmd.Flags().BoolP("disable-endpoint-per-replica", "", false, "Disable endpoint per replica")
	UpdateCmd.Flags().BoolP("mode-internal", "", false, "Set service as internal mode")
	UpdateCmd.Flags().BoolP("mode-external", "", false, "Set service as external mode")
	UpdateCmd.Flags().StringArrayP("cloud-providers", "", nil, "Update supported cloud providers (replaces existing)")
	UpdateCmd.Flags().StringArrayP("add-cloud-providers", "", nil, "Add cloud providers to existing list")
	UpdateCmd.Flags().StringP("instance-type-api-param", "", "", "Update API parameter name for instance type")
	UpdateCmd.Flags().StringP("api-params", "", "", "Update API parameters as JSON string (replaces existing)")
	UpdateCmd.Flags().StringP("add-api-params", "", "", "Add API parameters as JSON string to existing parameters")
	
	// Global integrations
	UpdateCmd.Flags().StringArrayP("integrations", "", nil, "Update Omnistrate integrations (replaces existing)")
	UpdateCmd.Flags().StringArrayP("add-integrations", "", nil, "Add Omnistrate integrations to existing list")
	
	// Compose version
	UpdateCmd.Flags().StringP("compose-version", "", "", "Update Docker Compose version")
}

// UpdateConfig represents the configuration for updating services
type UpdateConfig struct {
	ServiceName              string            `json:"serviceName,omitempty"`
	Image                    string            `json:"image,omitempty"`
	Ports                    []string          `json:"ports,omitempty"`
	Environment              []string          `json:"environment,omitempty"`
	Volumes                  []string          `json:"volumes,omitempty"`
	AddPorts                 []string          `json:"addPorts,omitempty"`
	AddEnvironment           []string          `json:"addEnvironment,omitempty"`
	AddVolumes               []string          `json:"addVolumes,omitempty"`
	RootVolumeSizeGi         *int              `json:"rootVolumeSizeGi,omitempty"`
	ReplicaCount             *int              `json:"replicaCount,omitempty"`
	ReplicaCountAPIParam     *string           `json:"replicaCountAPIParam,omitempty"`
	InstanceTypes            []InstanceType    `json:"instanceTypes,omitempty"`
	EnableMultiZone          *bool             `json:"enableMultiZone,omitempty"`
	EnableEndpointPerReplica *bool             `json:"enableEndpointPerReplica,omitempty"`
	ModeInternal             *bool             `json:"modeInternal,omitempty"`
	APIParams                []APIParam        `json:"apiParams,omitempty"`
	AddAPIParams             []APIParam        `json:"addApiParams,omitempty"`
	ActionHooks              []ActionHook      `json:"actionHooks,omitempty"`
	Autoscaling              *AutoscalingConfig `json:"autoscaling,omitempty"`
	BackupConfiguration      *BackupConfig     `json:"backupConfiguration,omitempty"`
}

func runUpdateCompose(cmd *cobra.Command, args []string) error {
	// Get input file
	inputFile, _ := cmd.Flags().GetString("input-file")
	outputFile, _ := cmd.Flags().GetString("output-file")
	
	// If no output file specified, overwrite input file
	if outputFile == "" {
		outputFile = inputFile
	}
	
	// Read existing compose file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file %s: %w", inputFile, err)
	}
	
	// Parse existing compose spec
	var spec ComposeSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}
	
	// Ensure services map is initialized
	if spec.Services == nil {
		spec.Services = make(map[string]Service)
	}
	
	// Get update flags
	servicePlanName, _ := cmd.Flags().GetString("service-plan-name")
	servicesJSON, _ := cmd.Flags().GetString("services")
	integrations, _ := cmd.Flags().GetStringArray("integrations")
	addIntegrations, _ := cmd.Flags().GetStringArray("add-integrations")
	composeVersion, _ := cmd.Flags().GetString("compose-version")
	
	// Update service plan name if provided
	if servicePlanName != "" {
		spec.ServicePlan.Name = servicePlanName
	}
	
	// Update compose version if provided
	if composeVersion != "" {
		spec.Version = composeVersion
	}
	
	// Update integrations
	if len(integrations) > 0 {
		spec.Integrations = integrations
	} else if len(addIntegrations) > 0 {
		// Add to existing integrations
		existingIntegrations := make(map[string]bool)
		for _, integration := range spec.Integrations {
			existingIntegrations[integration] = true
		}
		for _, integration := range addIntegrations {
			if !existingIntegrations[integration] {
				spec.Integrations = append(spec.Integrations, integration)
			}
		}
	}
	
	// Apply service updates
	var err2 error
	if servicesJSON != "" {
		err2 = applyServicesUpdatesFromJSON(servicesJSON, &spec)
	} else {
		err2 = applyServiceUpdatesFromFlags(cmd, &spec)
	}
	
	if err2 != nil {
		return err2
	}
	
	// Generate updated YAML
	yamlData, err := yaml.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}
	
	// Write the result
	return writeToFile(outputFile, yamlData)
}

func applyServicesUpdatesFromJSON(servicesJSON string, spec *ComposeSpec) error {
	var updateConfigs []UpdateConfig
	if err := json.Unmarshal([]byte(servicesJSON), &updateConfigs); err != nil {
		return fmt.Errorf("failed to parse services update JSON: %w", err)
	}
	
	for _, config := range updateConfigs {
		if config.ServiceName == "" {
			return fmt.Errorf("serviceName is required for all service updates")
		}
		
		// Get existing service or create new one
		service, exists := spec.Services[config.ServiceName]
		if !exists {
			service = Service{}
		}
		
		// Apply updates
		if err := applyServiceUpdate(&service, &config); err != nil {
			return fmt.Errorf("failed to update service '%s': %w", config.ServiceName, err)
		}
		
		spec.Services[config.ServiceName] = service
	}
	
	return nil
}

func applyServiceUpdatesFromFlags(cmd *cobra.Command, spec *ComposeSpec) error {
	serviceName, _ := cmd.Flags().GetString("service-name")
	
	// If no service name provided, no single service updates to apply
	if serviceName == "" {
		return nil
	}
	
	// Get existing service or create new one
	service, exists := spec.Services[serviceName]
	if !exists {
		service = Service{}
	}
	
	// Build update config from flags
	config := UpdateConfig{
		ServiceName: serviceName,
	}
	
	// Basic service properties
	if image, _ := cmd.Flags().GetString("image"); image != "" {
		config.Image = image
	}
	
	if ports, _ := cmd.Flags().GetStringArray("ports"); len(ports) > 0 {
		config.Ports = ports
	}
	if addPorts, _ := cmd.Flags().GetStringArray("add-ports"); len(addPorts) > 0 {
		config.AddPorts = addPorts
	}
	
	if environment, _ := cmd.Flags().GetStringArray("environment"); len(environment) > 0 {
		config.Environment = environment
	}
	if addEnvironment, _ := cmd.Flags().GetStringArray("add-environment"); len(addEnvironment) > 0 {
		config.AddEnvironment = addEnvironment
	}
	
	if volumes, _ := cmd.Flags().GetStringArray("volumes"); len(volumes) > 0 {
		config.Volumes = volumes
	}
	if addVolumes, _ := cmd.Flags().GetStringArray("add-volumes"); len(addVolumes) > 0 {
		config.AddVolumes = addVolumes
	}
	
	// Compute properties
	if rootVolumeSize, _ := cmd.Flags().GetInt("root-volume-size"); rootVolumeSize > 0 {
		config.RootVolumeSizeGi = &rootVolumeSize
	}
	if replicaCount, _ := cmd.Flags().GetInt("replica-count"); replicaCount > 0 {
		config.ReplicaCount = &replicaCount
	}
	if replicaCountAPIParam, _ := cmd.Flags().GetString("replica-count-api-param"); replicaCountAPIParam != "" {
		config.ReplicaCountAPIParam = &replicaCountAPIParam
	}
	
	// Capabilities
	if enableMultiZone, _ := cmd.Flags().GetBool("enable-multi-zone"); enableMultiZone {
		config.EnableMultiZone = &enableMultiZone
	}
	if disableMultiZone, _ := cmd.Flags().GetBool("disable-multi-zone"); disableMultiZone {
		falseVal := false
		config.EnableMultiZone = &falseVal
	}
	
	if enableEndpointPerReplica, _ := cmd.Flags().GetBool("enable-endpoint-per-replica"); enableEndpointPerReplica {
		config.EnableEndpointPerReplica = &enableEndpointPerReplica
	}
	if disableEndpointPerReplica, _ := cmd.Flags().GetBool("disable-endpoint-per-replica"); disableEndpointPerReplica {
		falseVal := false
		config.EnableEndpointPerReplica = &falseVal
	}
	
	if modeInternal, _ := cmd.Flags().GetBool("mode-internal"); modeInternal {
		config.ModeInternal = &modeInternal
	}
	if modeExternal, _ := cmd.Flags().GetBool("mode-external"); modeExternal {
		falseVal := false
		config.ModeInternal = &falseVal
	}
	
	// Cloud providers
	if cloudProviders, _ := cmd.Flags().GetStringArray("cloud-providers"); len(cloudProviders) > 0 {
		for _, provider := range cloudProviders {
			instanceType := InstanceType{CloudProvider: provider}
			if instanceTypeAPIParam, _ := cmd.Flags().GetString("instance-type-api-param"); instanceTypeAPIParam != "" {
				instanceType.APIParam = &instanceTypeAPIParam
			}
			config.InstanceTypes = append(config.InstanceTypes, instanceType)
		}
	}
	
	// API parameters
	if apiParamsJSON, _ := cmd.Flags().GetString("api-params"); apiParamsJSON != "" {
		var apiParams []APIParam
		if err := json.Unmarshal([]byte(apiParamsJSON), &apiParams); err != nil {
			return fmt.Errorf("failed to parse api-params JSON: %w", err)
		}
		config.APIParams = apiParams
	}
	if addAPIParamsJSON, _ := cmd.Flags().GetString("add-api-params"); addAPIParamsJSON != "" {
		var addAPIParams []APIParam
		if err := json.Unmarshal([]byte(addAPIParamsJSON), &addAPIParams); err != nil {
			return fmt.Errorf("failed to parse add-api-params JSON: %w", err)
		}
		config.AddAPIParams = addAPIParams
	}
	
	// Apply updates
	if err := applyServiceUpdate(&service, &config); err != nil {
		return fmt.Errorf("failed to update service '%s': %w", serviceName, err)
	}
	
	spec.Services[serviceName] = service
	return nil
}

func applyServiceUpdate(service *Service, config *UpdateConfig) error {
	// Update basic properties
	if config.Image != "" {
		service.Image = config.Image
	}
	
	// Update ports
	if len(config.Ports) > 0 {
		service.Ports = config.Ports
	}
	if len(config.AddPorts) > 0 {
		service.Ports = append(service.Ports, config.AddPorts...)
	}
	
	// Update environment
	if len(config.Environment) > 0 {
		service.Environment = config.Environment
	}
	if len(config.AddEnvironment) > 0 {
		service.Environment = append(service.Environment, config.AddEnvironment...)
	}
	
	// Update volumes
	if len(config.Volumes) > 0 {
		service.Volumes = config.Volumes
	}
	if len(config.AddVolumes) > 0 {
		service.Volumes = append(service.Volumes, config.AddVolumes...)
	}
	
	// Update compute configuration
	if config.RootVolumeSizeGi != nil || config.ReplicaCount != nil || config.ReplicaCountAPIParam != nil || len(config.InstanceTypes) > 0 {
		if service.Compute == nil {
			service.Compute = &Compute{}
		}
		
		if config.RootVolumeSizeGi != nil {
			service.Compute.RootVolumeSizeGi = config.RootVolumeSizeGi
		}
		if config.ReplicaCount != nil {
			service.Compute.ReplicaCount = config.ReplicaCount
		}
		if config.ReplicaCountAPIParam != nil {
			service.Compute.ReplicaCountAPIParam = config.ReplicaCountAPIParam
		}
		if len(config.InstanceTypes) > 0 {
			service.Compute.InstanceTypes = config.InstanceTypes
		}
	}
	
	// Update capabilities
	if config.EnableMultiZone != nil || config.EnableEndpointPerReplica != nil || config.Autoscaling != nil || config.BackupConfiguration != nil {
		if service.Capabilities == nil {
			service.Capabilities = &Capabilities{}
		}
		
		if config.EnableMultiZone != nil {
			service.Capabilities.EnableMultiZone = config.EnableMultiZone
		}
		if config.EnableEndpointPerReplica != nil {
			service.Capabilities.EnableEndpointPerReplica = config.EnableEndpointPerReplica
		}
		if config.Autoscaling != nil {
			service.Capabilities.Autoscaling = config.Autoscaling
		}
		if config.BackupConfiguration != nil {
			service.Capabilities.BackupConfiguration = config.BackupConfiguration
		}
	}
	
	// Update mode internal
	if config.ModeInternal != nil {
		service.ModeInternal = config.ModeInternal
	}
	
	// Update API parameters
	if len(config.APIParams) > 0 {
		service.APIParams = config.APIParams
	}
	if len(config.AddAPIParams) > 0 {
		// Add to existing API parameters, avoiding duplicates by key
		existingParams := make(map[string]bool)
		for _, param := range service.APIParams {
			existingParams[param.Key] = true
		}
		for _, param := range config.AddAPIParams {
			if !existingParams[param.Key] {
				service.APIParams = append(service.APIParams, param)
			}
		}
	}
	
	// Update action hooks
	if len(config.ActionHooks) > 0 {
		service.ActionHooks = config.ActionHooks
	}
	
	return nil
}
