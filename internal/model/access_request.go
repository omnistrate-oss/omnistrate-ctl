package model

// AccessRequest describes a service provider's request for deployment-cell access.
type AccessRequest struct {
	ID                   string `json:"id,omitempty"`
	ServiceProviderOrgID string `json:"serviceProviderOrgId,omitempty"`
	EnvironmentType      string `json:"environmentType,omitempty"`
	DeploymentCellID     string `json:"deploymentCellId,omitempty"`
	RequesterUserID      string `json:"requesterUserId,omitempty"`
	Role                 string `json:"role,omitempty"`
	Reason               string `json:"reason,omitempty"`
	CreatedAt            string `json:"createdAt,omitempty"`
	Status               string `json:"status,omitempty"`
	ApprovedBy           string `json:"approvedBy,omitempty"`
	ApprovedAt           string `json:"approvedAt,omitempty"`
	ExpiresAt            string `json:"expiresAt,omitempty"`
	RevokedBy            string `json:"revokedBy,omitempty"`
	RevokedAt            string `json:"revokedAt,omitempty"`
	ApprovalURL          string `json:"approvalUrl,omitempty"`
}

// AccessRequestList contains access requests visible to the authenticated organization.
type AccessRequestList struct {
	Requests []AccessRequest `json:"requests"`
}
