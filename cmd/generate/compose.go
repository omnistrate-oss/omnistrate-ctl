package generate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ComposeCmd represents the compose command under generate
var ComposeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Generate a Docker Compose spec with Omnistrate extensions for multiple services",
	Long: `Generate a Docker Compose spec with Omnistrate extensions based on provided parameters.
This command creates a compose file that follows the Omnistrate specification with x-omnistrate-* extensions.

The command supports two modes:
1. Multiple services mode: Use --services flag with JSON configuration to define multiple services
2. Legacy single service mode: Use individual flags for backward compatibility

Examples:

Multiple services with JSON:
  omnistrate-ctl generate compose \
    --service-plan-name "Multi-Service Stack" \
    --services '[
      {
        "name": "web",
        "image": "nginx:latest",
        "ports": ["80:80"],
        "replicaCount": 2,
        "enableMultiZone": true
      },
      {
        "name": "api",
        "image": "myapp:latest",
        "ports": ["8080:8080"],
        "environment": ["NODE_ENV=production"],
        "depends_on": ["database"]
      },
      {
        "name": "database",
        "image": "postgres:15",
        "environment": ["POSTGRES_DB=myapp"],
        "rootVolumeSizeGi": 50
      }
    ]'

Legacy single service:
  omnistrate-ctl generate compose \
    --service-plan-name "Single Service" \
    --service-name "web" \
    --image "nginx:latest" \
    --ports "80:80"`,
	RunE: runGenerateCompose,
}

type ComposeSpec struct {
	Version  string                 `yaml:"version"`
	Services map[string]Service     `yaml:"services"`
	Volumes  map[string]Volume      `yaml:"volumes,omitempty"`
	Networks map[string]Network     `yaml:"networks,omitempty"`
	Configs  map[string]Config      `yaml:"configs,omitempty"`
	Secrets  map[string]Secret      `yaml:"secrets,omitempty"`
	ServicePlan ServicePlan         `yaml:"x-omnistrate-service-plan"`
	Integrations []string           `yaml:"x-omnistrate-integrations,omitempty"`
	LoadBalancer *LoadBalancer      `yaml:"x-omnistrate-load-balancer,omitempty"`
	ImageRegistryAttributes map[string]ImageRegistryAuth `yaml:"x-omnistrate-image-registry-attributes,omitempty"`
}

type ServicePlan struct {
	Name string `yaml:"name"`
}

type Service struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment []string          `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty"`
	
	// Omnistrate-specific extensions
	Compute      *Compute      `yaml:"x-omnistrate-compute,omitempty"`
	Capabilities *Capabilities `yaml:"x-omnistrate-capabilities,omitempty"`
	APIParams    []APIParam    `yaml:"x-omnistrate-api-params,omitempty"`
	ActionHooks  []ActionHook  `yaml:"x-omnistrate-actionhooks,omitempty"`
	ModeInternal *bool         `yaml:"x-omnistrate-mode-internal,omitempty"`
}

type Compute struct {
	RootVolumeSizeGi      *int                    `yaml:"rootVolumeSizeGi,omitempty"`
	ReplicaCount          *int                    `yaml:"replicaCount,omitempty"`
	ReplicaCountAPIParam  *string                 `yaml:"replicaCountAPIParam,omitempty"`
	InstanceTypes         []InstanceType          `yaml:"instanceTypes,omitempty"`
}

type InstanceType struct {
	CloudProvider string  `yaml:"cloudProvider,omitempty"`
	APIParam      *string `yaml:"apiParam,omitempty"`
}

type Capabilities struct {
	EnableMultiZone         *bool               `yaml:"enableMultiZone,omitempty"`
	EnableEndpointPerReplica *bool              `yaml:"enableEndpointPerReplica,omitempty"`
	Autoscaling             *AutoscalingConfig  `yaml:"autoscaling,omitempty"`
	BackupConfiguration     *BackupConfig       `yaml:"backupConfiguration,omitempty"`
}

type AutoscalingConfig struct {
	MinReplicas *int `yaml:"minReplicas,omitempty"`
	MaxReplicas *int `yaml:"maxReplicas,omitempty"`
}

type BackupConfig struct {
	BackupPeriodInHours *int `yaml:"backupPeriodInHours,omitempty"`
}

type APIParam struct {
	Key          string      `yaml:"key"`
	Description  *string     `yaml:"description,omitempty"`
	Type         *string     `yaml:"type,omitempty"`
	DefaultValue interface{} `yaml:"defaultValue,omitempty"`
	Export       *bool       `yaml:"export,omitempty"`
}

type ActionHook struct {
	Scope   string `yaml:"scope"`
	Command string `yaml:"command"`
}

type Volume struct {
	Driver     *string           `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	External   *bool             `yaml:"external,omitempty"`
	Storage    *Storage          `yaml:"x-omnistrate-storage,omitempty"`
}

type Storage struct {
	AWS *AWSStorage `yaml:"aws,omitempty"`
	GCP *GCPStorage `yaml:"gcp,omitempty"`
}

type AWSStorage struct {
	InstanceStorageSizeGi *int    `yaml:"instanceStorageSizeGi,omitempty"`
	ClusterStorageType    *string `yaml:"clusterStorageType,omitempty"`
}

type GCPStorage struct {
	InstanceStorageSizeGi *int    `yaml:"instanceStorageSizeGi,omitempty"`
	StorageType           *string `yaml:"storageType,omitempty"`
}

type Network struct {
	Driver   *string           `yaml:"driver,omitempty"`
	External *bool             `yaml:"external,omitempty"`
	Options  map[string]string `yaml:"options,omitempty"`
}

type Config struct {
	File     *string `yaml:"file,omitempty"`
	External *bool   `yaml:"external,omitempty"`
}

type Secret struct {
	File     *string `yaml:"file,omitempty"`
	External *bool   `yaml:"external,omitempty"`
}

type LoadBalancer struct {
	HTTP  []HTTPRoute `yaml:"http,omitempty"`
	HTTPS []HTTPRoute `yaml:"https,omitempty"`
	TCP   []TCPRoute  `yaml:"tcp,omitempty"`
}

type HTTPRoute struct {
	Port   int    `yaml:"port"`
	Target string `yaml:"target"`
}

type TCPRoute struct {
	Port   int    `yaml:"port"`
	Target string `yaml:"target"`
}

type ImageRegistryAuth struct {
	Username *string `yaml:"username,omitempty"`
	Password *string `yaml:"password,omitempty"`
}

func init() {
	Cmd.AddCommand(ComposeCmd)
	
	// Service Plan flags
	ComposeCmd.Flags().StringP("service-plan-name", "n", "", "Name of the service plan (required)")
	ComposeCmd.MarkFlagRequired("service-plan-name")
	
	// Multiple services configuration
	ComposeCmd.Flags().StringP("services", "", "", "Services configuration as JSON string")
	
	// Legacy single service flags (for backward compatibility)
	ComposeCmd.Flags().StringP("service-name", "s", "", "Name of the main service (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringP("image", "i", "", "Docker image for the service (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringArray("ports", nil, "Port mappings (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringArray("environment", nil, "Environment variables (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringArray("volumes", nil, "Volume mounts (legacy, use --services for multiple services)")
	ComposeCmd.Flags().IntP("root-volume-size", "", 20, "Root volume size in GB (legacy, use --services for multiple services)")
	ComposeCmd.Flags().IntP("replica-count", "", 1, "Number of replicas (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringP("replica-count-api-param", "", "", "API parameter name for replica count (legacy, use --services for multiple services)")
	ComposeCmd.Flags().BoolP("enable-multi-zone", "", false, "Enable multi-zone deployment (legacy, use --services for multiple services)")
	ComposeCmd.Flags().BoolP("enable-endpoint-per-replica", "", false, "Enable endpoint per replica (legacy, use --services for multiple services)")
	ComposeCmd.Flags().BoolP("mode-internal", "", false, "Set service as internal mode (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringArrayP("cloud-providers", "", nil, "Supported cloud providers (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringP("instance-type-api-param", "", "", "API parameter name for instance type (legacy, use --services for multiple services)")
	ComposeCmd.Flags().StringP("api-params", "", "", "API parameters as JSON string (legacy, use --services for multiple services)")
	
	// Global integrations
	ComposeCmd.Flags().StringArrayP("integrations", "", nil, "Omnistrate integrations (omnistrateLogging, omnistrateMetrics)")
	
	// Output flags
	ComposeCmd.Flags().StringP("output-file", "f", "", "Output file path (default: stdout)")
	ComposeCmd.Flags().StringP("compose-version", "", "3.9", "Docker Compose version")
}

// ServiceConfig represents the configuration for a single service in JSON format
type ServiceConfig struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Ports        []string          `json:"ports,omitempty"`
	Environment  []string          `json:"environment,omitempty"`
	Volumes      []string          `json:"volumes,omitempty"`
	Networks     []string          `json:"networks,omitempty"`
	DependsOn    []string          `json:"depends_on,omitempty"`
	Command      []string          `json:"command,omitempty"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	
	// Omnistrate-specific configuration
	RootVolumeSizeGi         *int                    `json:"rootVolumeSizeGi,omitempty"`
	ReplicaCount             *int                    `json:"replicaCount,omitempty"`
	ReplicaCountAPIParam     *string                 `json:"replicaCountAPIParam,omitempty"`
	InstanceTypes            []InstanceType          `json:"instanceTypes,omitempty"`
	EnableMultiZone          *bool                   `json:"enableMultiZone,omitempty"`
	EnableEndpointPerReplica *bool                   `json:"enableEndpointPerReplica,omitempty"`
	ModeInternal             *bool                   `json:"modeInternal,omitempty"`
	APIParams                []APIParam              `json:"apiParams,omitempty"`
	ActionHooks              []ActionHook            `json:"actionHooks,omitempty"`
	Autoscaling              *AutoscalingConfig      `json:"autoscaling,omitempty"`
	BackupConfiguration      *BackupConfig           `json:"backupConfiguration,omitempty"`
}

func runGenerateCompose(cmd *cobra.Command, args []string) error {
	// Get common flags
	servicePlanName, _ := cmd.Flags().GetString("service-plan-name")
	servicesJSON, _ := cmd.Flags().GetString("services")
	integrations, _ := cmd.Flags().GetStringArray("integrations")
	outputFile, _ := cmd.Flags().GetString("output-file")
	composeVersion, _ := cmd.Flags().GetString("compose-version")
	
	// Build the compose spec
	spec := ComposeSpec{
		Version: composeVersion,
		ServicePlan: ServicePlan{
			Name: servicePlanName,
		},
		Services: make(map[string]Service),
	}
	
	// Add integrations if specified
	if len(integrations) > 0 {
		spec.Integrations = integrations
	}
	
	var err error
	
	// Check if using new JSON services format or legacy flags
	if servicesJSON != "" {
		err = parseServicesFromJSON(servicesJSON, &spec)
	} else {
		err = parseServiceFromLegacyFlags(cmd, &spec)
	}
	
	if err != nil {
		return err
	}
	
	// Validate that at least one service was configured
	if len(spec.Services) == 0 {
		return fmt.Errorf("no services configured. Use either --services flag with JSON configuration or legacy flags (--service-name, --image)")
	}
	
	// Generate YAML
	yamlData, err := yaml.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	
	// Output the result
	if outputFile != "" {
		return writeToFile(outputFile, yamlData)
	}
	
	fmt.Print(string(yamlData))
	return nil
}

func parseServicesFromJSON(servicesJSON string, spec *ComposeSpec) error {
	var serviceConfigs []ServiceConfig
	if err := json.Unmarshal([]byte(servicesJSON), &serviceConfigs); err != nil {
		return fmt.Errorf("failed to parse services JSON: %w", err)
	}
	
	for _, config := range serviceConfigs {
		if config.Name == "" {
			return fmt.Errorf("service name is required for all services")
		}
		if config.Image == "" {
			return fmt.Errorf("service image is required for service '%s'", config.Name)
		}
		
		service := Service{
			Image:       config.Image,
			Ports:       config.Ports,
			Environment: config.Environment,
			Volumes:     config.Volumes,
			Networks:    config.Networks,
			DependsOn:   config.DependsOn,
			Command:     config.Command,
			Entrypoint:  config.Entrypoint,
		}
		
		// Add compute configuration
		compute := &Compute{}
		hasCompute := false
		
		if config.RootVolumeSizeGi != nil && *config.RootVolumeSizeGi > 0 {
			compute.RootVolumeSizeGi = config.RootVolumeSizeGi
			hasCompute = true
		}
		if config.ReplicaCount != nil && *config.ReplicaCount > 0 {
			compute.ReplicaCount = config.ReplicaCount
			hasCompute = true
		}
		if config.ReplicaCountAPIParam != nil && *config.ReplicaCountAPIParam != "" {
			compute.ReplicaCountAPIParam = config.ReplicaCountAPIParam
			hasCompute = true
		}
		if len(config.InstanceTypes) > 0 {
			compute.InstanceTypes = config.InstanceTypes
			hasCompute = true
		}
		
		if hasCompute {
			service.Compute = compute
		}
		
		// Add capabilities
		capabilities := &Capabilities{}
		hasCapabilities := false
		
		if config.EnableMultiZone != nil {
			capabilities.EnableMultiZone = config.EnableMultiZone
			hasCapabilities = true
		}
		if config.EnableEndpointPerReplica != nil {
			capabilities.EnableEndpointPerReplica = config.EnableEndpointPerReplica
			hasCapabilities = true
		}
		if config.Autoscaling != nil {
			capabilities.Autoscaling = config.Autoscaling
			hasCapabilities = true
		}
		if config.BackupConfiguration != nil {
			capabilities.BackupConfiguration = config.BackupConfiguration
			hasCapabilities = true
		}
		
		if hasCapabilities {
			service.Capabilities = capabilities
		}
		
		// Set mode internal
		if config.ModeInternal != nil {
			service.ModeInternal = config.ModeInternal
		}
		
		// Set API parameters
		if len(config.APIParams) > 0 {
			service.APIParams = config.APIParams
		}
		
		// Set action hooks
		if len(config.ActionHooks) > 0 {
			service.ActionHooks = config.ActionHooks
		}
		
		spec.Services[config.Name] = service
	}
	
	return nil
}

func parseServiceFromLegacyFlags(cmd *cobra.Command, spec *ComposeSpec) error {
	// Get legacy flags
	serviceName, _ := cmd.Flags().GetString("service-name")
	image, _ := cmd.Flags().GetString("image")
	
	// Check if legacy flags are provided
	if serviceName == "" && image == "" {
		return nil // No legacy configuration provided
	}
	
	if serviceName == "" {
		return fmt.Errorf("service-name is required when using legacy flags")
	}
	if image == "" {
		return fmt.Errorf("image is required when using legacy flags")
	}
	
	ports, _ := cmd.Flags().GetStringArray("ports")
	environment, _ := cmd.Flags().GetStringArray("environment")
	volumes, _ := cmd.Flags().GetStringArray("volumes")
	
	rootVolumeSize, _ := cmd.Flags().GetInt("root-volume-size")
	replicaCount, _ := cmd.Flags().GetInt("replica-count")
	replicaCountAPIParam, _ := cmd.Flags().GetString("replica-count-api-param")
	
	enableMultiZone, _ := cmd.Flags().GetBool("enable-multi-zone")
	enableEndpointPerReplica, _ := cmd.Flags().GetBool("enable-endpoint-per-replica")
	modeInternal, _ := cmd.Flags().GetBool("mode-internal")
	
	cloudProviders, _ := cmd.Flags().GetStringArray("cloud-providers")
	instanceTypeAPIParam, _ := cmd.Flags().GetString("instance-type-api-param")
	
	apiParamsJSON, _ := cmd.Flags().GetString("api-params")
	
	// Build the service
	service := Service{
		Image:       image,
		Ports:       ports,
		Environment: environment,
		Volumes:     volumes,
	}
	
	// Add compute configuration
	compute := &Compute{}
	hasCompute := false
	
	if rootVolumeSize > 0 {
		compute.RootVolumeSizeGi = &rootVolumeSize
		hasCompute = true
	}
	if replicaCount > 0 {
		compute.ReplicaCount = &replicaCount
		hasCompute = true
	}
	if replicaCountAPIParam != "" {
		compute.ReplicaCountAPIParam = &replicaCountAPIParam
		hasCompute = true
	}
	
	// Add instance types for cloud providers
	if len(cloudProviders) > 0 {
		for _, provider := range cloudProviders {
			instanceType := InstanceType{
				CloudProvider: provider,
			}
			if instanceTypeAPIParam != "" {
				instanceType.APIParam = &instanceTypeAPIParam
			}
			compute.InstanceTypes = append(compute.InstanceTypes, instanceType)
		}
		hasCompute = true
	}
	
	if hasCompute {
		service.Compute = compute
	}
	
	// Add capabilities
	capabilities := &Capabilities{}
	hasCapabilities := false
	
	if enableMultiZone {
		capabilities.EnableMultiZone = &enableMultiZone
		hasCapabilities = true
	}
	if enableEndpointPerReplica {
		capabilities.EnableEndpointPerReplica = &enableEndpointPerReplica
		hasCapabilities = true
	}
	
	if hasCapabilities {
		service.Capabilities = capabilities
	}
	
	// Set mode internal
	if modeInternal {
		service.ModeInternal = &modeInternal
	}
	
	// Parse API parameters if provided
	if apiParamsJSON != "" {
		var apiParams []APIParam
		if err := json.Unmarshal([]byte(apiParamsJSON), &apiParams); err != nil {
			return fmt.Errorf("failed to parse api-params JSON: %w", err)
		}
		service.APIParams = apiParams
	}
	
	spec.Services[serviceName] = service
	return nil
}

func writeToFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()
	
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}
	
	fmt.Printf("Docker Compose spec written to %s\n", filename)
	return nil
}
