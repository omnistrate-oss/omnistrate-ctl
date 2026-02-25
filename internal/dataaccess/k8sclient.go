package dataaccess

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewK8sClientForDeploymentCell creates a Kubernetes client for a deployment cell using the Fleet API kubeconfig.
func NewK8sClientForDeploymentCell(ctx context.Context, token, deploymentCellID, role string) (*kubernetes.Clientset, error) {
	kubeConfig, err := GetKubeConfigForHostCluster(ctx, token, deploymentCellID, role)
	if err != nil {
		return nil, err
	}
	return NewK8sClientFromHostClusterKubeConfig(kubeConfig)
}

// NewK8sClientFromHostClusterKubeConfig builds a Kubernetes client from a Host Cluster kubeconfig result.
func NewK8sClientFromHostClusterKubeConfig(kubeConfig *openapiclientfleet.KubeConfigHostClusterResult) (*kubernetes.Clientset, error) {
	if kubeConfig == nil {
		return nil, fmt.Errorf("kubeconfig is nil")
	}

	apiServer := kubeConfig.GetApiServerEndpoint()
	if apiServer == "" {
		return nil, fmt.Errorf("api server endpoint is empty")
	}
	if !strings.HasPrefix(apiServer, "https://") {
		apiServer = "https://" + apiServer
	}

	caData, err := base64.StdEncoding.DecodeString(kubeConfig.GetCaDataBase64())
	if err != nil {
		return nil, fmt.Errorf("invalid CA data: %w", err)
	}
	certData, err := base64.StdEncoding.DecodeString(kubeConfig.GetClientCertificateDataBase64())
	if err != nil {
		return nil, fmt.Errorf("invalid client certificate data: %w", err)
	}
	keyData, err := base64.StdEncoding.DecodeString(kubeConfig.GetClientKeyDataBase64())
	if err != nil {
		return nil, fmt.Errorf("invalid client key data: %w", err)
	}

	cfg := &rest.Config{
		Host: apiServer,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   caData,
			CertData: certData,
			KeyData:  keyData,
		},
	}

	// Fallback to token auth if client certs are not provided.
	if kubeConfig.GetServiceAccountToken() != "" && (len(certData) == 0 || len(keyData) == 0) {
		cfg.BearerToken = kubeConfig.GetServiceAccountToken()
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %w", err)
	}
	return clientset, nil
}
