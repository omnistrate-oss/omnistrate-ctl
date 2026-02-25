package instance

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	terraformConfigMapNamespace      = "dataplane-agent"
	terraformProgressConfigMapPrefix = "terraform-progress-"
)

var terraformStateConfigMapPattern = regexp.MustCompile(`^tf-state-(.+)-instance-(.+)$`)

type terraformConfigMapIndex struct {
	instanceID      string
	instanceSuffix  string
	stateByResource map[string]*corev1.ConfigMap
	progress        []*corev1.ConfigMap
}

// k8sConnection holds both the clientset and rest config for k8s operations
type k8sConnection struct {
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
}

func loadTerraformConfigMapIndexForInstance(ctx context.Context, token string, instanceData *openapiclientfleet.ResourceInstance, instanceID string) (*terraformConfigMapIndex, *k8sConnection, error) {
	if instanceData == nil || instanceData.DeploymentCellID == nil || *instanceData.DeploymentCellID == "" {
		return nil, nil, fmt.Errorf("deployment cell ID not found for instance %s", instanceID)
	}

	deploymentCellID := *instanceData.DeploymentCellID
	kubeConfig, err := dataaccess.GetKubeConfigForHostCluster(ctx, token, deploymentCellID, "cluster-admin")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kubeconfig for deployment cell %s: %w", deploymentCellID, err)
	}

	conn, err := newK8sConnectionFromKubeConfigResult(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client for deployment cell %s: %w", deploymentCellID, err)
	}

	actualInstanceID := instanceID
	if id := instanceData.ConsumptionResourceInstanceResult.GetId(); id != "" {
		actualInstanceID = id
	}

	index, err := loadTerraformConfigMapIndex(ctx, conn.clientset, actualInstanceID)
	if err != nil {
		return nil, nil, err
	}

	return index, conn, nil
}

// newK8sConnectionFromKubeConfigResult writes a temp kubeconfig file from the API result and creates a k8s connection.
func newK8sConnectionFromKubeConfigResult(kubeConfig *openapiclientfleet.KubeConfigHostClusterResult) (*k8sConnection, error) {
	if kubeConfig == nil {
		return nil, fmt.Errorf("kubeconfig is nil")
	}

	clusterName := fmt.Sprintf("omnistrate-%s", kubeConfig.GetId())
	userName := fmt.Sprintf("omnistrate-%s", kubeConfig.GetUserName())
	contextName := clusterName

	apiServer := kubeConfig.GetApiServerEndpoint()
	if !strings.HasPrefix(apiServer, "https://") {
		apiServer = "https://" + apiServer
	}

	cluster := clientcmdapi.NewCluster()
	cluster.Server = apiServer

	caData, err := base64.StdEncoding.DecodeString(kubeConfig.GetCaDataBase64())
	if err == nil {
		cluster.CertificateAuthorityData = caData
	}

	authInfo := clientcmdapi.NewAuthInfo()
	certData, _ := base64.StdEncoding.DecodeString(kubeConfig.GetClientCertificateDataBase64())
	keyData, _ := base64.StdEncoding.DecodeString(kubeConfig.GetClientKeyDataBase64())
	if len(certData) > 0 && len(keyData) > 0 {
		authInfo.ClientCertificateData = certData
		authInfo.ClientKeyData = keyData
	}
	if kubeConfig.GetServiceAccountToken() != "" {
		authInfo.Token = kubeConfig.GetServiceAccountToken()
	}

	kubeConfigObj := clientcmdapi.NewConfig()
	kubeConfigObj.Clusters[clusterName] = cluster
	kubeConfigObj.AuthInfos[userName] = authInfo
	kubeConfigObj.Contexts[contextName] = &clientcmdapi.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	kubeConfigObj.CurrentContext = contextName

	// Write to a temp file
	tmpFile, err := os.CreateTemp("", "omnistrate-kubeconfig-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp kubeconfig file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := clientcmd.WriteToFile(*kubeConfigObj, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to write temp kubeconfig: %w", err)
	}

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: tmpPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build rest config from kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &k8sConnection{clientset: clientset, restConfig: restConfig}, nil
}

func loadTerraformConfigMapIndex(ctx context.Context, clientset kubernetes.Interface, instanceID string) (*terraformConfigMapIndex, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(terraformConfigMapNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps in namespace %s: %w", terraformConfigMapNamespace, err)
	}

	return newTerraformConfigMapIndex(instanceID, configMaps.Items), nil
}

func newTerraformConfigMapIndex(instanceID string, configMaps []corev1.ConfigMap) *terraformConfigMapIndex {
	index := &terraformConfigMapIndex{
		instanceID:      instanceID,
		instanceSuffix:  normalizeInstanceIDForConfigMap(instanceID),
		stateByResource: make(map[string]*corev1.ConfigMap),
		progress:        []*corev1.ConfigMap{},
	}

	for i := range configMaps {
		cm := &configMaps[i]
		if strings.HasPrefix(cm.Name, terraformProgressConfigMapPrefix) {
			index.progress = append(index.progress, cm)
			continue
		}

		matches := terraformStateConfigMapPattern.FindStringSubmatch(cm.Name)
		if len(matches) != 3 {
			continue
		}

		resourceID := matches[1]
		instanceSuffix := matches[2]
		if instanceSuffix != index.instanceSuffix && instanceSuffix != index.instanceID {
			continue
		}

		index.stateByResource[resourceID] = cm
	}

	return index
}

func (index *terraformConfigMapIndex) terraformDataForResource(resourceID string) *TerraformData {
	terraformData := &TerraformData{
		Files: make(map[string]string),
		Logs:  make(map[string]string),
	}

	if index == nil || resourceID == "" {
		return terraformData
	}

	if stateConfigMap, ok := index.stateByResource[resourceID]; ok {
		for key, value := range stateConfigMap.Data {
			terraformData.Files[normalizeTerraformFileKey(key)] = value
		}
	}

	if progressConfigMap := index.findBestProgressConfigMap(resourceID); progressConfigMap != nil {
		for key, value := range progressConfigMap.Data {
			terraformData.Logs[normalizeTerraformLogKey(key)] = value
		}
	}

	return terraformData
}

func (index *terraformConfigMapIndex) findBestProgressConfigMap(resourceID string) *corev1.ConfigMap {
	var best *corev1.ConfigMap
	bestScore := 0

	for _, cm := range index.progress {
		score := scoreTerraformProgressConfigMap(cm, index.instanceID, index.instanceSuffix, resourceID)
		if score == 0 {
			continue
		}

		if best == nil || score > bestScore || (score == bestScore && isConfigMapNewer(cm, best)) {
			best = cm
			bestScore = score
		}
	}

	return best
}

func scoreTerraformProgressConfigMap(cm *corev1.ConfigMap, instanceID, instanceSuffix, resourceID string) int {
	if cm == nil {
		return 0
	}

	score := 0
	hasResource := false
	hasInstance := false

	if resourceID != "" && strings.Contains(cm.Name, resourceID) {
		score += 5
		hasResource = true
	}

	if instanceID != "" && (strings.Contains(cm.Name, instanceID) || (instanceSuffix != "" && strings.Contains(cm.Name, instanceSuffix))) {
		score += 4
		hasInstance = true
	}

	labelScore, labelHasResource, labelHasInstance := matchScoreFromMap(cm.Labels, instanceID, instanceSuffix, resourceID)
	score += labelScore
	hasResource = hasResource || labelHasResource
	hasInstance = hasInstance || labelHasInstance

	annotationScore, annotationHasResource, annotationHasInstance := matchScoreFromMap(cm.Annotations, instanceID, instanceSuffix, resourceID)
	score += annotationScore
	hasResource = hasResource || annotationHasResource
	hasInstance = hasInstance || annotationHasInstance

	dataScore, dataHasResource, dataHasInstance := matchScoreFromData(cm.Data, instanceID, instanceSuffix, resourceID)
	score += dataScore
	hasResource = hasResource || dataHasResource
	hasInstance = hasInstance || dataHasInstance

	if hasResource && hasInstance {
		score += 3
	}

	return score
}

func matchScoreFromMap(values map[string]string, instanceID, instanceSuffix, resourceID string) (int, bool, bool) {
	score := 0
	hasResource := false
	hasInstance := false

	for key, value := range values {
		lowerKey := strings.ToLower(key)

		if resourceID != "" && (strings.Contains(value, resourceID) || strings.Contains(key, resourceID)) {
			hasResource = true
			if strings.Contains(lowerKey, "resource") {
				score += 3
			} else {
				score += 1
			}
		}

		if instanceID != "" && (strings.Contains(value, instanceID) || strings.Contains(key, instanceID)) {
			hasInstance = true
			if strings.Contains(lowerKey, "instance") {
				score += 3
			} else {
				score += 1
			}
			continue
		}

		if instanceSuffix != "" && (strings.Contains(value, instanceSuffix) || strings.Contains(key, instanceSuffix)) {
			hasInstance = true
			if strings.Contains(lowerKey, "instance") {
				score += 2
			} else {
				score += 1
			}
		}
	}

	return score, hasResource, hasInstance
}

func matchScoreFromData(data map[string]string, instanceID, instanceSuffix, resourceID string) (int, bool, bool) {
	score := 0
	hasResource := false
	hasInstance := false

	for key, value := range data {
		if resourceID != "" && strings.Contains(key, resourceID) {
			hasResource = true
			score++
		}

		if instanceID != "" && (strings.Contains(key, instanceID) || (instanceSuffix != "" && strings.Contains(key, instanceSuffix))) {
			hasInstance = true
			score++
		}

		if len(value) > 512 {
			continue
		}

		if resourceID != "" && strings.Contains(value, resourceID) {
			hasResource = true
			score++
		}

		if instanceID != "" && (strings.Contains(value, instanceID) || (instanceSuffix != "" && strings.Contains(value, instanceSuffix))) {
			hasInstance = true
			score++
		}
	}

	return score, hasResource, hasInstance
}

func isConfigMapNewer(candidate, existing *corev1.ConfigMap) bool {
	if candidate == nil || existing == nil {
		return false
	}
	return candidate.CreationTimestamp.After(existing.CreationTimestamp.Time)
}

func normalizeInstanceIDForConfigMap(instanceID string) string {
	return strings.TrimPrefix(instanceID, "instance-")
}

func normalizeTerraformLogKey(key string) string {
	if strings.HasPrefix(key, "log/") {
		return key
	}
	return "log/" + key
}

func normalizeTerraformFileKey(key string) string {
	return key
}
