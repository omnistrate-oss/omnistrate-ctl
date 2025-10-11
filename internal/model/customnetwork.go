package model

type CustomNetwork struct {
	CustomNetworkID              string `json:"custom_network_id"`
	CustomNetworkName            string `json:"custom_network_name"`
	CloudProvider                string `json:"cloud_provider"`
	Region                       string `json:"region"`
	CIDR                         string `json:"cidr"`
	OwningOrgID                  string `json:"owning_org_id"`
	OwningOrgName                string `json:"owning_org_name"`
	AwsAccountID                 string `json:"aws_account_id"`
	CloudProviderNativeNetworkId string `json:"cloud_provider_native_network_id"`
	GcpProjectID                 string `json:"gcp_project_id"`
	GcpProjectNumber             string `json:"gcp_project_number"`
	HostClusterID                string `json:"host_cluster_id"`
	Status                       string `json:"status"`
}
