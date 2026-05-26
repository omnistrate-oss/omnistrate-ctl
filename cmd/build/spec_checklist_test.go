package build

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSchemaURLFromSpec_YAMLLanguageServerDirective(t *testing.T) {
	spec := []byte(`# yaml-language-server: $schema=https://api.omnistrate.cloud/2022-09-01-00/schema/compose-spec-schema.json
services:
  api:
    image: nginx
`)

	require.Equal(t, composeSpecSchemaURL, schemaURLFromSpec(spec))
}

func TestSchemaURLFromSpec_IntelliJDirective(t *testing.T) {
	spec := []byte(`# $schema: https://api.omnistrate.cloud/2022-09-01-00/schema/service-spec-schema.json
name: test
services: []
`)

	require.Equal(t, serviceSpecSchemaURL, schemaURLFromSpec(spec))
}

func TestBuildSpecChecklist_ComposeSpecUsesPlatformActions(t *testing.T) {
	spec := []byte(`# yaml-language-server: $schema=https://api.omnistrate.cloud/2022-09-01-00/schema/compose-spec-schema.json
x-omnistrate-service-plan:
  name: dev
  tenancyType: OMNISTRATE_DEDICATED_TENANCY
  features:
    dashboard: {}
    metrics#INTERNAL: {}
services:
  api:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - source: api-data
        target: /data
        x-omnistrate-storage:
          aws:
            instanceStorageType: AWS::EBS_GP3
            instanceStorageSizeGi: 100
    x-omnistrate-compute:
      replicaCount: 2
    x-omnistrate-capabilities:
      autoscaling:
        minReplicas:
          Type: 0
          IntVal: 1
          StrVal: "1"
`)

	items, err := buildSpecChecklist(spec, DockerComposeSpecType)

	require.NoError(t, err)
	labels := specChecklistLabels(items)
	require.Contains(t, labels, "Resolving plan dev")
	require.Contains(t, labels, "Setting tenancy to dedicated")
	require.Contains(t, labels, "Enabling Dashboard feature (customer-facing)")
	require.Contains(t, labels, "Enabling Metrics feature (internal)")
	require.Contains(t, labels, "Preparing resource api")
	require.Contains(t, labels, "Configuring container image nginx:latest")
	require.Contains(t, labels, "Configuring networking")
	require.Contains(t, labels, "Configuring port 8080:80")
	require.Contains(t, labels, "Configuring compute")
	require.Contains(t, labels, "Configuring replica count to 2")
	require.Contains(t, labels, "Configuring storage")
	require.Contains(t, labels, "Configuring storage aws")
	require.Contains(t, labels, "Enabling autoscaling")
}

func TestBuildSpecChecklist_ServicePlanSpecShowsDeploymentAccountsAndResources(t *testing.T) {
	spec := []byte(`# yaml-language-server: $schema=https://api.omnistrate.cloud/2022-09-01-00/schema/service-spec-schema.json
name: dev
deployment:
  hostedDeployment:
    AwsAccountId: "123"
features:
  audit:
    audience: internal
services:
  - name: redis
    helmChartConfiguration:
      chartName: redis
      chartVersion: 1.0.0
      chartRepoName: bitnami
      chartRepoURL: https://charts.bitnami.com/bitnami
    compute:
      replicaCount: 1
      instanceTypes:
        - apiParam: instanceType
          cloudProvider: aws
    apiParameters:
      - key: password
        name: Password
`)

	items, err := buildSpecChecklist(spec, ServicePlanSpecType)

	require.NoError(t, err)
	labels := specChecklistLabels(items)
	require.Contains(t, labels, "Resolving plan dev")
	require.Contains(t, labels, "Configuring hosted deployment")
	require.Contains(t, labels, "Connecting AWS account 123")
	require.Contains(t, labels, "Enabling Audit feature (internal)")
	require.Contains(t, labels, "Preparing resource redis")
	require.Contains(t, labels, "Configuring Helm chart redis 1.0.0")
	require.Contains(t, labels, "Configuring compute")
	require.Contains(t, labels, "Configuring replica count to 1")
	require.Contains(t, labels, "Configuring compute instance type parameter instanceType for AWS")
	require.Contains(t, labels, "Configuring deployment parameters")
	require.Contains(t, labels, "Configuring deployment parameter password")
	requireResourceSubItems(t, items, "Preparing resource redis")
}

func TestBuildSpecChecklist_DistributionExtensionStartsChecklistAndReplacesDeploymentModel(t *testing.T) {
	spec := []byte(`# yaml-language-server: $schema=https://api.omnistrate.cloud/2022-09-01-00/schema/service-spec-schema.json
name: dev
deployment:
  byoaDeployment:
    AwsAccountId: "123"
services: []
`)

	items, err := buildSpecChecklistWithOptions(spec, ServicePlanSpecType, specChecklistOptions{extendDistribution: true})

	require.NoError(t, err)
	require.NotEmpty(t, items)
	require.Equal(t, "Extending distribution to BYOA deployment model", items[0].Label)
	labels := specChecklistLabels(items)
	require.Contains(t, labels, "Connecting AWS account 123")
	require.NotContains(t, labels, "Configuring BYOA deployment")
}

func TestBuildSpecChecklist_FormatsVarReferencesAsDeploymentParameters(t *testing.T) {
	spec := []byte(`# yaml-language-server: $schema=https://api.omnistrate.cloud/2022-09-01-00/schema/service-spec-schema.json
name: parametrized
services:
  - name: api
    image: repo:$var.imageTag
    helmChartConfiguration:
      chartName: redis
      chartVersion: $var.chartVersion
    compute:
      replicaCount: $var.replicaCount
      rootVolumeSizeGi: $var.rootVolumeSize
      instanceTypes:
        - name: $var.instanceType
          cloudProvider: aws
    storage:
      data:
        instanceStorageSizeGi: $var.storageSize
        instanceStorageType: $var.storageType
    network:
      ports:
        - $var.port
    capabilities:
      httpReverseProxy:
        targetPort: $var.proxyPort
`)

	items, err := buildSpecChecklist(spec, ServicePlanSpecType)

	require.NoError(t, err)
	labels := specChecklistLabels(items)
	require.Contains(t, labels, "Configuring container image with deployment parameter imageTag")
	require.Contains(t, labels, "Configuring Helm chart with deployment parameter chartVersion")
	require.Contains(t, labels, "Configuring replica count with deployment parameter replicaCount")
	require.Contains(t, labels, "Configuring root volume with deployment parameter rootVolumeSize")
	require.Contains(t, labels, "Configuring compute instance type with deployment parameter instanceType")
	require.Contains(t, labels, "Configuring storage size with deployment parameter storageSize")
	require.Contains(t, labels, "Configuring storage type with deployment parameter storageType")
	require.Contains(t, labels, "Configuring port with deployment parameter port")
	require.Contains(t, labels, "Configuring HTTP reverse proxy with deployment parameter proxyPort")
}

func TestIsAllowedSpecSchemaURL(t *testing.T) {
	require.True(t, isAllowedSpecSchemaURL(composeSpecSchemaURL))
	require.True(t, isAllowedSpecSchemaURL(serviceSpecSchemaURL))
	require.False(t, isAllowedSpecSchemaURL("https://example.com/schema.json"))
}

func TestRandomChecklistDelayDoesNotExceedMaximum(t *testing.T) {
	for i := 0; i < 100; i++ {
		delay := randomChecklistDelay()
		require.Greater(t, delay, time.Duration(0))
		require.LessOrEqual(t, delay, time.Duration(maxChecklistDelayMillis)*time.Millisecond)
	}
}

func specChecklistLabels(items []specChecklistItem) []string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, strings.TrimSpace(item.Label))
	}
	sort.Strings(labels)
	return labels
}

func requireResourceSubItems(t *testing.T, items []specChecklistItem, resourceLabel string) {
	t.Helper()
	resourceIndex := -1
	for i, item := range items {
		if item.Label == resourceLabel {
			resourceIndex = i
			break
		}
	}
	require.NotEqual(t, -1, resourceIndex)
	require.Greater(t, len(items), resourceIndex+1)
	require.Greater(t, items[resourceIndex+1].Depth, items[resourceIndex].Depth)
}
