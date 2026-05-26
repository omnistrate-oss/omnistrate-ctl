package build

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"gopkg.in/yaml.v3"
)

const (
	composeSpecSchemaURL = "https://api.omnistrate.cloud/2022-09-01-00/schema/compose-spec-schema.json"
	serviceSpecSchemaURL = "https://api.omnistrate.cloud/2022-09-01-00/schema/service-spec-schema.json"

	maxChecklistDelayMillis = 500
)

var (
	schemaDirectiveRE     = regexp.MustCompile(`(?m)^\s*#\s*(?:yaml-language-server:\s*)?\$schema\s*(?:=|:)\s*(\S+)\s*$`)
	deploymentParameterRE = regexp.MustCompile(`\$var\.([A-Za-z_][A-Za-z0-9_]*)`)
)

type specChecklistItem struct {
	Label string
	Depth int
}

type specChecklistOptions struct {
	extendDistribution bool
}

type specBuildProgress struct {
	sm                utils.SpinnerManager
	featureSpinners   []*utils.Spinner
	finalizingSpinner *utils.Spinner
}

func addSpecBuildChecklist(sm utils.SpinnerManager, fileData []byte, specType string, options specChecklistOptions) *specBuildProgress {
	progress := &specBuildProgress{sm: sm}
	if sm == nil {
		return progress
	}

	items, err := buildSpecChecklistWithOptions(fileData, specType, options)
	if err == nil {
		for _, item := range items {
			spinner := sm.AddSpinner(indentChecklistLabel(item))
			spinner.Pending()
			progress.featureSpinners = append(progress.featureSpinners, spinner)
		}
	}
	return progress
}

func (p *specBuildProgress) StreamFeatureChecks(ctx context.Context) error {
	if p == nil {
		return nil
	}
	for _, spinner := range p.featureSpinners {
		if spinner == nil {
			continue
		}
		spinner.Start()
		if err := waitChecklistDelay(ctx); err != nil {
			return err
		}
		spinner.Complete()
	}
	return nil
}

func (p *specBuildProgress) StartFinalizing() *utils.Spinner {
	if p == nil || p.sm == nil {
		return nil
	}
	if p.finalizingSpinner == nil {
		p.finalizingSpinner = p.sm.AddSpinner("Finalizing build")
	}
	p.finalizingSpinner.Start()
	return p.finalizingSpinner
}

func (p *specBuildProgress) FinalizingSpinner() *utils.Spinner {
	if p == nil {
		return nil
	}
	return p.finalizingSpinner
}

func (p *specBuildProgress) CompleteFinalizing() {
	if p == nil || p.finalizingSpinner == nil {
		return
	}
	p.finalizingSpinner.Complete()
}

func waitChecklistDelay(ctx context.Context) error {
	delay := randomChecklistDelay()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func randomChecklistDelay() time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(maxChecklistDelayMillis))
	if err != nil {
		return 100 * time.Millisecond
	}
	return time.Duration(n.Int64()+1) * time.Millisecond
}

func indentChecklistLabel(item specChecklistItem) string {
	if item.Depth <= 0 {
		return item.Label
	}
	return strings.Repeat("  ", item.Depth) + item.Label
}

func buildSpecChecklist(fileData []byte, specType string) ([]specChecklistItem, error) {
	return buildSpecChecklistWithOptions(fileData, specType, specChecklistOptions{})
}

func buildSpecChecklistWithOptions(fileData []byte, specType string, options specChecklistOptions) ([]specChecklistItem, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(fileData, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, nil
	}

	schemaURL := schemaURLFromSpec(fileData)
	if schemaURL == "" {
		schemaURL = defaultSchemaURLForSpecType(specType)
	}
	if !isAllowedSpecSchemaURL(schemaURL) {
		schemaURL = defaultSchemaURLForSpecType(specType)
	}
	if specType == "" {
		if schemaURL == serviceSpecSchemaURL {
			specType = ServicePlanSpecType
		} else {
			specType = DockerComposeSpecType
		}
	}

	if specType == ServicePlanSpecType {
		return servicePlanSpecChecklist(root.Content[0], options), nil
	}
	return composeSpecChecklist(root.Content[0], options), nil
}

func schemaURLFromSpec(fileData []byte) string {
	matches := schemaDirectiveRE.FindSubmatch(fileData)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(string(matches[1]))
}

func defaultSchemaURLForSpecType(specType string) string {
	if specType == ServicePlanSpecType {
		return serviceSpecSchemaURL
	}
	return composeSpecSchemaURL
}

func isAllowedSpecSchemaURL(schemaURL string) bool {
	return schemaURL == composeSpecSchemaURL || schemaURL == serviceSpecSchemaURL
}

func composeSpecChecklist(root *yaml.Node, options specChecklistOptions) []specChecklistItem {
	var items []specChecklistItem

	if plan := mappingValue(root, "x-omnistrate-service-plan"); plan != nil {
		addPlanChecklist(plan, &items, options)
	}
	if lb := mappingValue(root, "x-omnistrate-load-balancer"); lb != nil {
		addLoadBalancerChecklist(lb, &items, 0)
	}
	addTopLevelComposeCollections(root, &items)

	services := mappingValue(root, "services")
	if services != nil && services.Kind == yaml.MappingNode {
		for _, entry := range mappingEntries(services) {
			name := entry.key
			if name == "" {
				name = "unnamed"
			}
			addResourceChecklist(name, entry.value, DockerComposeSpecType, &items)
		}
	}

	return dedupeSpecChecklistItems(items)
}

func servicePlanSpecChecklist(root *yaml.Node, options specChecklistOptions) []specChecklistItem {
	var items []specChecklistItem

	addPlanChecklist(root, &items, options)
	if lb := mappingValue(root, "loadBalancers"); lb != nil {
		addLoadBalancerChecklist(lb, &items, 0)
	}
	if sharedFileSystems := mappingValue(root, "sharedFileSystems"); sharedFileSystems != nil {
		addSharedFileSystemsChecklist(sharedFileSystems, &items, 0)
	}
	addWorkflowChecklist(root, &items, 0)

	services := mappingValue(root, "services")
	if services != nil && services.Kind == yaml.SequenceNode {
		for i, service := range services.Content {
			name := mappingScalar(service, "name")
			if name == "" {
				name = fmt.Sprintf("resource %d", i+1)
			}
			addResourceChecklist(name, service, ServicePlanSpecType, &items)
		}
	}

	return dedupeSpecChecklistItems(items)
}

func addPlanChecklist(plan *yaml.Node, items *[]specChecklistItem, options specChecklistOptions) {
	deployment := mappingValue(plan, "deployment")
	if options.extendDistribution {
		appendChecklist(items, 0, "Extending distribution to %s deployment model", deploymentModelLabel(deployment))
		addDeploymentCloudAccountsChecklist(deployment, items, 1)
	}
	if name := mappingScalar(plan, "name"); name != "" {
		appendChecklist(items, 0, "Resolving plan %s", name)
	}
	if tenancy := mappingScalar(plan, "tenancyType"); tenancy != "" {
		appendChecklist(items, 0, "Setting tenancy to %s", humanizeIdentifier(tenancy))
	}
	if deployment != nil && !options.extendDistribution {
		addDeploymentChecklist(deployment, items, 0)
	}
	if features := mappingValue(plan, "features"); features != nil {
		addFeatureChecklist(features, items, 0)
	}
	if pricing := mappingValue(plan, "pricing"); pricing != nil {
		addPricingChecklist(pricing, items, 0)
	}
	if metering := mappingValue(plan, "metering"); metering != nil {
		addMeteringChecklist(metering, items, 0)
	}
	if providers := mappingValue(plan, "billingProviders"); providers != nil {
		addBillingProvidersChecklist(providers, items, 0)
	}
	if limit := mappingScalar(plan, "maxNumberOfInstancesAllowed"); limit != "" {
		appendChecklist(items, 0, "Limiting plan to %s instance(s)", limit)
	}
	if boolValue(plan, "validPaymentMethodRequired") {
		appendChecklist(items, 0, "Requiring a valid payment method")
	}
	if boolValue(plan, "enableDeletionProtection") {
		appendChecklist(items, 0, "Enabling deletion protection")
	}
	if billingProductID := mappingScalar(plan, "billingProductID"); billingProductID != "" {
		appendChecklist(items, 0, "Linking billing product %s", billingProductID)
	}
}

func addDeploymentChecklist(deployment *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, deploymentType := range deploymentChecklistTypes() {
		config := mappingValue(deployment, deploymentType.key)
		if config == nil {
			continue
		}
		appendChecklist(items, depth, "Configuring %s deployment", deploymentType.label)
		addCloudAccountChecklist(config, items, depth+1)
	}
}

func addDeploymentCloudAccountsChecklist(deployment *yaml.Node, items *[]specChecklistItem, depth int) {
	if deployment == nil {
		return
	}
	for _, deploymentType := range deploymentChecklistTypes() {
		config := mappingValue(deployment, deploymentType.key)
		if config == nil {
			continue
		}
		addCloudAccountChecklist(config, items, depth)
	}
}

func deploymentModelLabel(deployment *yaml.Node) string {
	for _, deploymentType := range deploymentChecklistTypes() {
		if mappingValue(deployment, deploymentType.key) != nil {
			return deploymentType.label
		}
	}
	return "unknown"
}

func deploymentChecklistTypes() []struct {
	key   string
	label string
} {
	return []struct {
		key   string
		label string
	}{
		{"hostedDeployment", "hosted"},
		{"byoaDeployment", "BYOA"},
		{"onPremDeployment", "on-prem"},
		{"onPremCopilotDeployment", "on-prem copilot"},
	}
}

func addCloudAccountChecklist(config *yaml.Node, items *[]specChecklistItem, depth int) {
	if awsAccountID := mappingScalar(config, "AwsAccountId"); awsAccountID != "" {
		appendChecklist(items, depth, "Connecting AWS account %s", awsAccountID)
	}
	if roleArn := mappingScalar(config, "AWSBootstrapRoleAccountArn"); roleArn != "" {
		appendChecklist(items, depth, "Using AWS bootstrap role %s", roleArn)
	}

	gcpProjectID := mappingScalar(config, "GcpProjectId")
	gcpProjectNumber := mappingScalar(config, "GcpProjectNumber")
	switch {
	case gcpProjectID != "" && gcpProjectNumber != "":
		appendChecklist(items, depth, "Connecting GCP project %s (%s)", gcpProjectID, gcpProjectNumber)
	case gcpProjectID != "":
		appendChecklist(items, depth, "Connecting GCP project %s", gcpProjectID)
	case gcpProjectNumber != "":
		appendChecklist(items, depth, "Connecting GCP project number %s", gcpProjectNumber)
	}
	if serviceAccount := mappingScalar(config, "GcpServiceAccountEmail"); serviceAccount != "" {
		appendChecklist(items, depth, "Using GCP service account %s", serviceAccount)
	}

	azureSubscriptionID := mappingScalar(config, "AzureSubscriptionId")
	azureTenantID := mappingScalar(config, "AzureTenantId")
	switch {
	case azureSubscriptionID != "" && azureTenantID != "":
		appendChecklist(items, depth, "Connecting Azure subscription %s (tenant %s)", azureSubscriptionID, azureTenantID)
	case azureSubscriptionID != "":
		appendChecklist(items, depth, "Connecting Azure subscription %s", azureSubscriptionID)
	case azureTenantID != "":
		appendChecklist(items, depth, "Using Azure tenant %s", azureTenantID)
	}

	if nebiusTenantID := mappingScalar(config, "NebiusTenantId"); nebiusTenantID != "" {
		appendChecklist(items, depth, "Connecting Nebius tenant %s", nebiusTenantID)
	}
	if ociTenancyID := mappingScalar(config, "OCITenancyId"); ociTenancyID != "" {
		appendChecklist(items, depth, "Connecting OCI tenancy %s", ociTenancyID)
	}
	if ociDomainID := mappingScalar(config, "OCIDomainId"); ociDomainID != "" {
		appendChecklist(items, depth, "Using OCI domain %s", ociDomainID)
	}
}

func addFeatureChecklist(features *yaml.Node, items *[]specChecklistItem, depth int) {
	if features.Kind != yaml.MappingNode {
		return
	}
	for _, entry := range mappingEntries(features) {
		name := featureDisplayName(entry.key)
		if name == "" {
			continue
		}
		appendChecklist(items, depth, "Enabling %s feature (%s)", name, featureAudience(entry.key, entry.value))
	}
}

func addPricingChecklist(pricing *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, node := range collectionNodes(pricing) {
		dimension := mappingScalar(node, "dimension")
		if dimension == "" {
			dimension = "pricing"
		}
		appendChecklist(items, depth, "Configuring %s pricing", humanizeIdentifier(dimension))
	}
}

func addMeteringChecklist(metering *yaml.Node, items *[]specChecklistItem, depth int) {
	switch {
	case mappingScalar(metering, "s3BucketArn") != "":
		appendChecklist(items, depth, "Exporting metering to S3")
	case mappingScalar(metering, "gcsBucketName") != "":
		appendChecklist(items, depth, "Exporting metering to GCS")
	default:
		appendChecklist(items, depth, "Configuring metering")
	}
}

func addBillingProvidersChecklist(providers *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, node := range collectionNodes(providers) {
		name := mappingScalar(node, "name")
		if name == "" {
			name = "billing provider"
		}
		appendChecklist(items, depth, "Configuring %s billing", name)
	}
}

func addLoadBalancerChecklist(loadBalancers *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, protocol := range []string{"https", "tcp"} {
		configs := mappingValue(loadBalancers, protocol)
		for _, node := range collectionNodes(configs) {
			name := mappingScalar(node, "name")
			if name == "" {
				name = strings.ToUpper(protocol)
			}
			appendChecklist(items, depth, "Configuring %s load balancer %s", strings.ToUpper(protocol), name)
			addLoadBalancerRoutesChecklist(node, items, depth+1)
		}
	}
}

func addLoadBalancerRoutesChecklist(node *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, port := range collectionNodes(mappingValue(node, "ports")) {
		backendPort := mappingScalar(port, "backendPort")
		ingressPort := mappingScalar(port, "ingressPort")
		if backendPort != "" && ingressPort != "" {
			appendValuesChecklist(items, depth, "load balancer port", []string{ingressPort, backendPort}, "Routing port %s to backend port %s", ingressPort, backendPort)
		}
	}
	for _, path := range collectionNodes(mappingValue(node, "paths")) {
		route := mappingScalar(path, "path")
		backendPort := mappingScalar(path, "backendPort")
		resource := mappingScalar(path, "associatedResourceKey")
		if route == "" {
			route = "/"
		}
		switch {
		case resource != "" && backendPort != "":
			appendValuesChecklist(items, depth, "load balancer route", []string{route, backendPort}, "Routing %s to %s on port %s", route, resource, backendPort)
		case backendPort != "":
			appendValuesChecklist(items, depth, "load balancer route", []string{route, backendPort}, "Routing %s to backend port %s", route, backendPort)
		default:
			appendValueChecklist(items, depth, "load balancer route", route, "Routing %s", route)
		}
	}
}

func addSharedFileSystemsChecklist(sharedFileSystems *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, node := range collectionNodes(sharedFileSystems) {
		name := mappingScalar(node, "name")
		if name == "" {
			name = "shared file system"
		}
		appendChecklist(items, depth, "Configuring shared file system %s", name)
	}
}

func addWorkflowChecklist(root *yaml.Node, items *[]specChecklistItem, depth int) {
	if workflows := mappingValue(root, "systemWorkflows"); workflows != nil {
		for _, key := range []string{"backup", "restore", "deleteBackup"} {
			if mappingValue(workflows, key) != nil {
				appendChecklist(items, depth, "Registering %s workflow", humanizeIdentifier(key))
			}
		}
	}
	if workflows := mappingValue(root, "customWorkflows"); workflows != nil {
		for _, node := range collectionNodes(workflows) {
			displayName := mappingScalar(node, "displayName")
			if displayName == "" {
				displayName = mappingScalar(node, "verb")
			}
			if displayName == "" {
				displayName = "custom"
			}
			appendChecklist(items, depth, "Registering %s workflow", displayName)
		}
	}
}

func addTopLevelComposeCollections(root *yaml.Node, items *[]specChecklistItem) {
	for _, entry := range []struct {
		key    string
		action string
	}{
		{"volumes", "Configuring shared volume"},
		{"networks", "Configuring network"},
		{"configs", "Configuring config"},
		{"secrets", "Configuring secret"},
	} {
		node := mappingValue(root, entry.key)
		if node == nil || node.Kind != yaml.MappingNode {
			continue
		}
		for _, value := range mappingEntries(node) {
			appendChecklist(items, 0, "%s %s", entry.action, value.key)
		}
	}
}

func addResourceChecklist(name string, resource *yaml.Node, specType string, items *[]specChecklistItem) {
	appendChecklist(items, 0, "Preparing resource %s", name)
	addResourceRuntimeChecklist(resource, specType, items, 1)
	addComputeChecklist(resourceComputeNode(resource, specType), items, 1)
	addStorageChecklist(resource, specType, items, 1)
	addNetworkChecklist(resource, specType, items, 1)
	addCapabilitiesChecklist(resourceCapabilitiesNode(resource, specType), items, 1)
	addResourceInputsChecklist(resource, specType, items, 1)
	addResourceConfigChecklist(resource, items, 1)
	addResourceLifecycleChecklist(resource, specType, items, 1)
	addResourceSecurityChecklist(resource, specType, items, 1)
}

func addResourceRuntimeChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	if image := mappingScalar(resource, "image"); image != "" {
		appendValueChecklist(items, depth, "container image", image, "Configuring container image %s", image)
	}
	if build := mappingValue(resource, "build"); build != nil {
		switch build.Kind {
		case yaml.ScalarNode:
			appendValueChecklist(items, depth, "container build", build.Value, "Configuring container build from %s", build.Value)
		case yaml.MappingNode:
			context := mappingScalar(build, "context")
			if context == "" {
				context = "."
			}
			appendValueChecklist(items, depth, "container build", context, "Configuring container build from %s", context)
		}
	}

	if specType == ServicePlanSpecType {
		if helm := mappingValue(resource, "helmChartConfiguration"); helm != nil {
			chartName := mappingScalar(helm, "chartName")
			chartVersion := mappingScalar(helm, "chartVersion")
			if chartName == "" {
				chartName = "Helm chart"
			}
			if chartVersion != "" {
				appendValuesChecklist(items, depth, "Helm chart", []string{chartName, chartVersion}, "Configuring Helm chart %s %s", chartName, chartVersion)
			} else {
				appendValueChecklist(items, depth, "Helm chart", chartName, "Configuring Helm chart %s", chartName)
			}
		}
		if mappingValue(resource, "operatorCRDConfiguration") != nil {
			appendChecklist(items, depth, "Configuring operator CRD")
		}
		if mappingValue(resource, "terraformConfigurations") != nil {
			appendChecklist(items, depth, "Configuring Terraform")
		}
		if kustomize := mappingValue(resource, "kustomizeConfiguration"); kustomize != nil {
			path := mappingScalar(kustomize, "kustomizePath")
			if path == "" {
				path = "overlay"
			}
			appendValueChecklist(items, depth, "Kustomize", path, "Configuring Kustomize %s", path)
		}
		if mappingValue(resource, "agentConfiguration") != nil {
			appendChecklist(items, depth, "Configuring agent runtime")
		}
		if mappingValue(resource, "containerImagesRegistryCopyConfiguration") != nil {
			appendChecklist(items, depth, "Configuring container image registry copy")
		}
	}

	if mappingValue(resource, "command") != nil || mappingValue(resource, "args") != nil || mappingValue(resource, "entrypoint") != nil {
		appendChecklist(items, depth, "Configuring startup command")
	}
}

func resourceComputeNode(resource *yaml.Node, specType string) *yaml.Node {
	if specType == ServicePlanSpecType {
		return mappingValue(resource, "compute")
	}
	return mappingValue(resource, "x-omnistrate-compute")
}

func addComputeChecklist(compute *yaml.Node, items *[]specChecklistItem, depth int) {
	if compute == nil {
		return
	}
	var section []specChecklistItem
	if replicaCount := mappingScalar(compute, "replicaCount"); replicaCount != "" {
		appendValueChecklist(&section, depth+1, "replica count", replicaCount, "Configuring replica count to %s", replicaCount)
	}
	if apiParam := mappingScalar(compute, "replicaCountAPIParam"); apiParam != "" {
		appendChecklist(&section, depth+1, "Configuring replica count parameter %s", apiParam)
	}
	if architecture := mappingScalar(compute, "cpuArchitecture"); architecture != "" {
		appendValueChecklist(&section, depth+1, "CPU architecture", architecture, "Configuring %s CPU architecture", architecture)
	}
	if rootVolumeSize := mappingScalar(compute, "rootVolumeSizeGi"); rootVolumeSize != "" {
		appendValueChecklist(&section, depth+1, "root volume", rootVolumeSize, "Configuring root volume to %s GiB", rootVolumeSize)
	}
	addComputeUnitChecklist(mappingValue(compute, "reservations"), "reservations", &section, depth+1)
	addComputeUnitChecklist(mappingValue(compute, "limits"), "limits", &section, depth+1)
	addInstanceTypesChecklist(mappingValue(compute, "instanceTypes"), &section, depth+1)
	appendChecklistSection(items, depth, "Configuring compute", section)
}

func addComputeUnitChecklist(unit *yaml.Node, label string, items *[]specChecklistItem, depth int) {
	if unit == nil {
		return
	}
	cpu := quantityValue(mappingValue(unit, "cpu"))
	memory := quantityValue(mappingValue(unit, "memory"))
	switch {
	case cpu != "" && memory != "":
		appendValuesChecklist(items, depth, "compute "+label, []string{cpu, memory}, "Configuring compute %s for %s CPU and %s memory", label, cpu, memory)
	case cpu != "":
		appendValueChecklist(items, depth, "compute "+label, cpu, "Configuring compute %s for %s CPU", label, cpu)
	case memory != "":
		appendValueChecklist(items, depth, "compute "+label, memory, "Configuring compute %s for %s memory", label, memory)
	}
}

func addInstanceTypesChecklist(instanceTypes *yaml.Node, items *[]specChecklistItem, depth int) {
	for _, node := range collectionNodes(instanceTypes) {
		name := mappingScalar(node, "name")
		apiParam := mappingScalar(node, "apiParam")
		cloudProvider := strings.ToUpper(mappingScalar(node, "cloudProvider"))
		platform := mappingScalar(node, "platform")
		if name == "" && apiParam == "" {
			continue
		}
		switch {
		case apiParam != "" && cloudProvider != "":
			appendChecklist(items, depth, "Configuring compute instance type parameter %s for %s", apiParam, cloudProvider)
		case apiParam != "":
			appendChecklist(items, depth, "Configuring compute instance type parameter %s", apiParam)
		case name != "" && deploymentParameterName(name) != "":
			appendValueChecklist(items, depth, "compute instance type", name, "Configuring compute instance type %s", name)
		case cloudProvider != "" && platform != "":
			appendValuesChecklist(items, depth, "compute instance type", []string{name, platform}, "Configuring compute instance type %s %s on %s", cloudProvider, name, platform)
		case cloudProvider != "":
			appendValueChecklist(items, depth, "compute instance type", name, "Configuring compute instance type %s %s", cloudProvider, name)
		default:
			appendValueChecklist(items, depth, "compute instance type", name, "Configuring compute instance type %s", name)
		}
	}
}

func addStorageChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	var section []specChecklistItem
	var storage *yaml.Node
	if specType == ServicePlanSpecType {
		storage = mappingValue(resource, "storage")
	} else {
		storage = mappingValue(resource, "x-omnistrate-storage")
	}
	addStorageEntriesChecklist(storage, &section, depth+1)

	if volumes := mappingValue(resource, "volumes"); volumes != nil {
		for _, volume := range collectionNodes(volumes) {
			if volume.Kind == yaml.ScalarNode {
				appendValueChecklist(&section, depth+1, "volume mount", volume.Value, "Configuring volume mount %s", volume.Value)
				continue
			}
			target := mappingScalar(volume, "target")
			source := mappingScalar(volume, "source")
			switch {
			case source != "" && target != "":
				appendValuesChecklist(&section, depth+1, "volume mount", []string{source, target}, "Configuring volume mount %s at %s", source, target)
			case target != "":
				appendValueChecklist(&section, depth+1, "volume mount", target, "Configuring volume mount at %s", target)
			}
			addStorageEntriesChecklist(mappingValue(volume, "x-omnistrate-storage"), &section, depth+2)
		}
	}
	appendChecklistSection(items, depth, "Configuring storage", section)
}

func addStorageEntriesChecklist(storage *yaml.Node, items *[]specChecklistItem, depth int) {
	if storage == nil {
		return
	}
	for _, entry := range mappingEntries(storage) {
		name := entry.key
		if name == "" {
			name = "storage"
		}
		appendChecklist(items, depth, "Configuring storage %s", name)
		if entry.value.Kind == yaml.MappingNode {
			addStorageParamsChecklist(entry.value, items, depth+1)
			continue
		}
		for _, params := range collectionNodes(entry.value) {
			addStorageParamsChecklist(params, items, depth+1)
		}
	}
}

func addStorageParamsChecklist(params *yaml.Node, items *[]specChecklistItem, depth int) {
	if params == nil || params.Kind != yaml.MappingNode {
		return
	}
	if size := mappingScalar(params, "instanceStorageSizeGi"); size != "" {
		appendValueChecklist(items, depth, "storage size", size, "Configuring storage size to %s GiB", size)
	}
	if storageType := mappingScalar(params, "instanceStorageType"); storageType != "" {
		appendValueChecklist(items, depth, "storage type", storageType, "Configuring storage type %s", storageType)
	}
	if storageTypeParam := mappingScalar(params, "instanceStorageTypeAPIParam"); storageTypeParam != "" {
		appendChecklist(items, depth, "Configuring storage type parameter %s", storageTypeParam)
	}
	if sizeParam := mappingScalar(params, "instanceStorageSizeGiAPIParam"); sizeParam != "" {
		appendChecklist(items, depth, "Configuring storage size parameter %s", sizeParam)
	}
}

func addNetworkChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	var section []specChecklistItem
	if ports := mappingValue(resource, "ports"); ports != nil {
		for _, port := range collectionNodes(ports) {
			if port.Kind == yaml.ScalarNode {
				appendValueChecklist(&section, depth+1, "port", port.Value, "Configuring port %s", port.Value)
				continue
			}
			target := mappingScalar(port, "target")
			published := mappingScalar(port, "published")
			switch {
			case target != "" && published != "":
				appendValuesChecklist(&section, depth+1, "port", []string{target, published}, "Configuring port %s as %s", target, published)
			case target != "":
				appendValueChecklist(&section, depth+1, "port", target, "Configuring port %s", target)
			}
		}
	}

	if specType == ServicePlanSpecType {
		if network := mappingValue(resource, "network"); network != nil {
			if ports := mappingValue(network, "ports"); ports != nil {
				for _, port := range collectionNodes(ports) {
					if port.Value != "" {
						appendValueChecklist(&section, depth+1, "port", port.Value, "Configuring port %s", port.Value)
					}
				}
			}
		}
		if endpoints := mappingValue(resource, "endpointConfiguration"); endpoints != nil {
			for _, entry := range mappingEntries(endpoints) {
				appendChecklist(&section, depth+1, "Configuring endpoint %s", entry.key)
			}
		}
	} else if networks := mappingValue(resource, "networks"); networks != nil {
		appendChecklist(&section, depth+1, "Configuring network attachments")
	}
	appendChecklistSection(items, depth, "Configuring networking", section)
}

func resourceCapabilitiesNode(resource *yaml.Node, specType string) *yaml.Node {
	if specType == ServicePlanSpecType {
		return mappingValue(resource, "capabilities")
	}
	return mappingValue(resource, "x-omnistrate-capabilities")
}

func addCapabilitiesChecklist(capabilities *yaml.Node, items *[]specChecklistItem, depth int) {
	if capabilities == nil || capabilities.Kind != yaml.MappingNode {
		return
	}
	var section []specChecklistItem
	if mappingValue(capabilities, "autoscaling") != nil {
		appendChecklist(&section, depth+1, "Enabling autoscaling")
	}
	if mappingValue(capabilities, "backupConfiguration") != nil {
		appendChecklist(&section, depth+1, "Configuring backups")
	}
	if dns := mappingValue(capabilities, "customDNS"); dns != nil {
		name := mappingScalar(dns, "Name")
		if name == "" {
			name = mappingScalar(dns, "name")
		}
		if name != "" {
			appendValueChecklist(&section, depth+1, "custom DNS", name, "Configuring custom DNS %s", name)
		} else {
			appendChecklist(&section, depth+1, "Configuring custom DNS")
		}
	}
	if proxy := mappingValue(capabilities, "httpReverseProxy"); proxy != nil {
		if targetPort := mappingScalar(proxy, "targetPort"); targetPort != "" {
			appendValueChecklist(&section, depth+1, "HTTP reverse proxy", targetPort, "Configuring HTTP reverse proxy to port %s", targetPort)
		} else {
			appendChecklist(&section, depth+1, "Configuring HTTP reverse proxy")
		}
	}
	if mappingValue(capabilities, "serverlessConfiguration") != nil {
		appendChecklist(&section, depth+1, "Enabling serverless configuration")
	}
	if boolValue(capabilities, "enableMultiZone") {
		appendChecklist(&section, depth+1, "Enabling multi-zone placement")
	}
	if boolValue(capabilities, "enableCustomZone") {
		appendChecklist(&section, depth+1, "Enabling custom zone placement")
	}
	if boolValue(capabilities, "enableEndpointPerReplica") {
		appendChecklist(&section, depth+1, "Configuring endpoint per replica")
	}
	if networkType := mappingScalar(capabilities, "networkType"); networkType != "" {
		appendValueChecklist(&section, depth+1, "network type", networkType, "Configuring network type %s", networkType)
	}
	if boolValue(capabilities, "enableStableEgressIP") {
		appendChecklist(&section, depth+1, "Enabling stable egress IP")
	}
	if boolValue(capabilities, "enableClusterLoadBalancer") {
		appendChecklist(&section, depth+1, "Enabling cluster load balancer")
	}
	if boolValue(capabilities, "enableNodeLoadBalancer") {
		appendChecklist(&section, depth+1, "Enabling node load balancer")
	}
	if mappingValue(capabilities, "serviceAccountPolicies") != nil {
		appendChecklist(&section, depth+1, "Configuring service account policies")
	}
	if mappingValue(capabilities, "nodeWarmPool") != nil {
		appendChecklist(&section, depth+1, "Configuring node warm pool")
	}
	if mappingValue(capabilities, "sidecars") != nil {
		appendChecklist(&section, depth+1, "Configuring sidecars")
	}
	appendChecklistSection(items, depth, "Configuring resource capabilities", section)
}

func addResourceInputsChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	var params *yaml.Node
	if specType == ServicePlanSpecType {
		params = mappingValue(resource, "apiParameters")
	} else {
		params = mappingValue(resource, "x-omnistrate-api-params")
	}
	var paramSection []specChecklistItem
	for _, param := range collectionNodes(params) {
		key := mappingScalar(param, "key")
		if key == "" {
			key = mappingScalar(param, "name")
		}
		if key == "" {
			key = "input"
		}
		appendChecklist(&paramSection, depth+1, "Configuring deployment parameter %s", key)
	}
	appendChecklistSection(items, depth, "Configuring deployment parameters", paramSection)

	if env := mappingValue(resource, "environment"); env != nil {
		appendChecklist(items, depth, "Injecting %d environment variable(s)", collectionLen(env))
	}
	if env := mappingValue(resource, "environmentVariables"); env != nil {
		appendChecklist(items, depth, "Injecting %d environment variable(s)", collectionLen(env))
	}
	if envFile := mappingValue(resource, "env_file"); envFile != nil {
		appendChecklist(items, depth, "Loading environment file(s)")
	}
}

func addResourceConfigChecklist(resource *yaml.Node, items *[]specChecklistItem, depth int) {
	if configs := mappingValue(resource, "configs"); configs != nil {
		appendChecklist(items, depth, "Configuring %d config item(s)", collectionLen(configs))
	}
	if secrets := mappingValue(resource, "secrets"); secrets != nil {
		appendChecklist(items, depth, "Configuring %d secret item(s)", collectionLen(secrets))
	}
}

func addResourceLifecycleChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	if mappingValue(resource, "healthcheck") != nil {
		appendChecklist(items, depth, "Configuring health check")
	}

	var hooks *yaml.Node
	if specType == ServicePlanSpecType {
		hooks = mappingValue(resource, "actionHooks")
	} else {
		hooks = mappingValue(resource, "x-omnistrate-actionhooks")
	}
	for _, hook := range collectionNodes(hooks) {
		hookType := mappingScalar(hook, "Type")
		if hookType == "" {
			hookType = "lifecycle"
		}
		appendChecklist(items, depth, "Registering %s action hook", humanizeIdentifier(hookType))
	}

	if specType == ServicePlanSpecType {
		if mappingValue(resource, "jobResourceConfiguration") != nil {
			appendChecklist(items, depth, "Configuring job execution")
		}
		if depends := mappingValue(resource, "dependsOn"); depends != nil {
			appendChecklist(items, depth, "Linking resource dependencies")
		}
	} else {
		if mappingValue(resource, "x-omnistrate-job-config") != nil {
			appendChecklist(items, depth, "Configuring job execution")
		}
		if depends := mappingValue(resource, "depends_on"); depends != nil {
			appendChecklist(items, depth, "Linking service dependencies")
		}
	}
}

func addResourceSecurityChecklist(resource *yaml.Node, specType string, items *[]specChecklistItem, depth int) {
	if boolValue(resource, "privileged") {
		appendChecklist(items, depth, "Configuring privileged access")
	}
	if specType == ServicePlanSpecType {
		if boolValue(resource, "internal") {
			appendChecklist(items, depth, "Marking resource as internal")
		}
		if proxyType := mappingScalar(resource, "proxyType"); proxyType != "" {
			appendValueChecklist(items, depth, "proxy type", proxyType, "Configuring proxy type %s", proxyType)
		}
	} else {
		if boolValue(resource, "x-omnistrate-mode-internal") {
			appendChecklist(items, depth, "Marking resource as internal")
		}
		if proxyType := mappingScalar(resource, "x-omnistrate-proxy-type"); proxyType != "" {
			appendValueChecklist(items, depth, "proxy type", proxyType, "Configuring proxy type %s", proxyType)
		}
	}
}

func appendChecklist(items *[]specChecklistItem, depth int, format string, args ...interface{}) {
	label := fmt.Sprintf(format, args...)
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	*items = append(*items, specChecklistItem{Label: label, Depth: depth})
}

func appendValueChecklist(items *[]specChecklistItem, depth int, thing string, value string, fallbackFormat string, args ...interface{}) {
	if parameter := deploymentParameterName(value); parameter != "" {
		appendChecklist(items, depth, "Configuring %s with deployment parameter %s", thing, parameter)
		return
	}
	appendChecklist(items, depth, "%s", fmt.Sprintf(fallbackFormat, args...))
}

func appendValuesChecklist(items *[]specChecklistItem, depth int, thing string, values []string, fallbackFormat string, args ...interface{}) {
	for _, value := range values {
		if parameter := deploymentParameterName(value); parameter != "" {
			appendChecklist(items, depth, "Configuring %s with deployment parameter %s", thing, parameter)
			return
		}
	}
	appendChecklist(items, depth, "%s", fmt.Sprintf(fallbackFormat, args...))
}

func deploymentParameterName(value string) string {
	matches := deploymentParameterRE.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func appendChecklistSection(items *[]specChecklistItem, depth int, label string, section []specChecklistItem) {
	if len(section) == 0 {
		return
	}
	appendChecklist(items, depth, "%s", label)
	*items = append(*items, section...)
}

func featureDisplayName(featureKey string) string {
	name := strings.TrimSpace(featureKey)
	if delimiter := strings.IndexAny(name, "#:"); delimiter >= 0 {
		name = strings.TrimSpace(name[:delimiter])
	}
	name = strings.Join(trimAudienceTokens(name), " ")
	return humanizeIdentifier(name)
}

func featureAudience(featureKey string, node *yaml.Node) string {
	if audience := audienceFromText(featureKey); audience != "" {
		return audience
	}
	if audience := audienceFromNode(node, 0); audience != "" {
		return audience
	}
	return "customer-facing"
}

func audienceFromNode(node *yaml.Node, depth int) string {
	if node == nil || depth > 2 {
		return ""
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return audienceFromText(node.Value)
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if audience := audienceFromNode(child, depth+1); audience != "" {
				return audience
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := strings.ToLower(strings.TrimSpace(node.Content[i].Value))
			value := node.Content[i+1]
			if key == "internal" {
				if strings.EqualFold(scalarValue(value), "true") {
					return "internal"
				}
				if strings.EqualFold(scalarValue(value), "false") {
					return "customer-facing"
				}
			}
			if key == "customerfacing" || key == "customer-facing" || key == "customer_facing" {
				if strings.EqualFold(scalarValue(value), "true") {
					return "customer-facing"
				}
				if strings.EqualFold(scalarValue(value), "false") {
					return "internal"
				}
			}
			if key == "audience" || key == "visibility" || key == "scope" || key == "access" || key == "mode" || key == "type" {
				if audience := audienceFromText(scalarValue(value)); audience != "" {
					return audience
				}
			}
			if audience := audienceFromNode(value, depth+1); audience != "" {
				return audience
			}
		}
	}
	return ""
}

func audienceFromText(value string) string {
	for _, token := range audienceTokens(value) {
		switch token {
		case "INTERNAL", "PRIVATE":
			return "internal"
		case "CUSTOMER", "CUSTOMERFACING", "PUBLIC", "EXTERNAL":
			return "customer-facing"
		}
	}
	return ""
}

func trimAudienceTokens(value string) []string {
	tokens := audienceTokens(value)
	for len(tokens) > 0 && isFeatureAudienceToken(tokens[0]) {
		tokens = tokens[1:]
	}
	for len(tokens) > 0 && isFeatureAudienceToken(tokens[len(tokens)-1]) {
		tokens = tokens[:len(tokens)-1]
	}
	if len(tokens) == 0 {
		return []string{value}
	}
	for i, token := range tokens {
		tokens[i] = strings.ToLower(token)
	}
	return tokens
}

func audienceTokens(value string) []string {
	normalized := strings.NewReplacer(
		"#", " ",
		":", " ",
		"_", " ",
		"-", " ",
		".", " ",
		"/", " ",
	).Replace(strings.ToUpper(strings.TrimSpace(value)))
	return strings.Fields(normalized)
}

func isFeatureAudienceToken(token string) bool {
	switch token {
	case "INTERNAL", "PRIVATE", "CUSTOMER", "CUSTOMERFACING", "FACING", "PUBLIC", "EXTERNAL":
		return true
	default:
		return false
	}
}

func dedupeSpecChecklistItems(items []specChecklistItem) []specChecklistItem {
	seen := make(map[string]struct{}, len(items))
	result := make([]specChecklistItem, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%d:%s", item.Depth, item.Label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func mappingEntries(node *yaml.Node) []struct {
	key   string
	value *yaml.Node
} {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	entries := make([]struct {
		key   string
		value *yaml.Node
	}, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		entries = append(entries, struct {
			key   string
			value *yaml.Node
		}{key: strings.TrimSpace(node.Content[i].Value), value: node.Content[i+1]})
	}
	return entries
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func mappingScalar(node *yaml.Node, key string) string {
	return scalarValue(mappingValue(node, key))
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func boolValue(node *yaml.Node, key string) bool {
	value := strings.ToLower(mappingScalar(node, key))
	return value == "true" || value == "yes" || value == "on"
}

func collectionNodes(node *yaml.Node) []*yaml.Node {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.SequenceNode:
		return node.Content
	case yaml.MappingNode:
		nodes := make([]*yaml.Node, 0, len(node.Content)/2)
		for i := 1; i < len(node.Content); i += 2 {
			nodes = append(nodes, node.Content[i])
		}
		return nodes
	default:
		return nil
	}
}

func collectionLen(node *yaml.Node) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case yaml.SequenceNode:
		return len(node.Content)
	case yaml.MappingNode:
		return len(node.Content) / 2
	default:
		return 1
	}
}

func quantityValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return scalarValue(node)
	}
	if format := mappingScalar(node, "Format"); format != "" {
		return format
	}
	return ""
}

func humanizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	upper := strings.ToUpper(value)
	switch upper {
	case "OMNISTRATE DEDICATED TENANCY":
		return "dedicated"
	case "OMNISTRATE MULTI TENANCY":
		return "multi-tenant"
	}
	parts := strings.Fields(strings.ToLower(value))
	for i, part := range parts {
		if part == "id" || part == "dns" || part == "api" || part == "http" || part == "tcp" {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return value
	}
	return strings.Join(parts, " ")
}
