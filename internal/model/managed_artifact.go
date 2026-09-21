package model

// ManagedArtifactReleasePolicy describes how a Base Amenities template selects
// a managed artifact release for an environment and cloud provider.
type ManagedArtifactReleasePolicy struct {
	EnvironmentType        string `json:"environmentType"`
	CloudProvider          string `json:"cloudProvider"`
	AutoUpgrade            bool   `json:"autoUpgrade"`
	PreferredBundleVersion string `json:"preferredBundleVersion,omitempty"`
	EffectiveBundleVersion string `json:"effectiveBundleVersion,omitempty"`
}

// UpdateManagedArtifactReleasePolicyRequest updates release selection behavior.
type UpdateManagedArtifactReleasePolicyRequest struct {
	AutoUpgrade            bool   `json:"autoUpgrade"`
	PreferredBundleVersion string `json:"preferredBundleVersion,omitempty"`
}

// ManagedArtifactTargetStatus summarizes synchronization state across targets.
type ManagedArtifactTargetStatus struct {
	TotalTargetCount      int `json:"totalTargetCount"`
	PendingTargetCount    int `json:"pendingTargetCount"`
	InProgressTargetCount int `json:"inProgressTargetCount"`
	ReadyTargetCount      int `json:"readyTargetCount"`
	FailedTargetCount     int `json:"failedTargetCount"`
	SkippedTargetCount    int `json:"skippedTargetCount"`
}

// ManagedArtifactReleaseArtifact is a chart or image used by a managed amenity.
type ManagedArtifactReleaseArtifact struct {
	AmenityName    string   `json:"amenityName"`
	ArtifactKey    string   `json:"artifactKey"`
	Name           string   `json:"name,omitempty"`
	Type           string   `json:"type"`
	Version        string   `json:"version"`
	SourceRef      string   `json:"sourceRef"`
	SourceDigest   string   `json:"sourceDigest,omitempty"`
	SourceChecksum string   `json:"sourceChecksum,omitempty"`
	RelativePath   string   `json:"relativePath"`
	Platforms      []string `json:"platforms,omitempty"`
}

// ManagedArtifactRelease is a published managed artifact bundle.
type ManagedArtifactRelease struct {
	BundleVersion      string                           `json:"bundleVersion"`
	ReleaseSequence    int64                            `json:"releaseSequence"`
	ReleasedAt         string                           `json:"releasedAt"`
	CreatedAt          string                           `json:"createdAt,omitempty"`
	LastTransitionTime string                           `json:"lastTransitionTime,omitempty"`
	AmenityCount       int                              `json:"amenityCount"`
	ArtifactCount      int                              `json:"artifactCount"`
	TargetStatus       ManagedArtifactTargetStatus      `json:"targetStatus"`
	Artifacts          []ManagedArtifactReleaseArtifact `json:"artifacts,omitempty"`
}

// ManagedArtifactReleaseList contains one page of managed releases.
type ManagedArtifactReleaseList struct {
	Releases      []ManagedArtifactRelease `json:"releases"`
	NextPageToken string                   `json:"nextPageToken,omitempty"`
}

// ManagedArtifactTarget identifies a provisioner synchronization target.
type ManagedArtifactTarget struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	CloudProvider string `json:"cloudProvider,omitempty"`
	Region        string `json:"region,omitempty"`
	AccountID     string `json:"accountId,omitempty"`
}

// ManagedArtifactSyncArtifact is the synchronization result for one artifact.
type ManagedArtifactSyncArtifact struct {
	AmenityName     string `json:"amenityName"`
	ArtifactKey     string `json:"artifactKey"`
	Version         string `json:"version"`
	Status          string `json:"status"`
	CompletedAt     string `json:"completedAt,omitempty"`
	FailureCategory string `json:"failureCategory,omitempty"`
	FailureMessage  string `json:"failureMessage,omitempty"`
}

// ManagedArtifactSync describes synchronization to one provisioner target.
type ManagedArtifactSync struct {
	ID                     string                        `json:"id"`
	BundleVersion          string                        `json:"bundleVersion"`
	Target                 ManagedArtifactTarget         `json:"target"`
	Status                 string                        `json:"status"`
	ArtifactCount          int                           `json:"artifactCount"`
	CompletedArtifactCount int                           `json:"completedArtifactCount"`
	FailedArtifactCount    int                           `json:"failedArtifactCount"`
	FailureCategory        string                        `json:"failureCategory,omitempty"`
	FailureMessage         string                        `json:"failureMessage,omitempty"`
	LastTransitionTime     string                        `json:"lastTransitionTime,omitempty"`
	CreatedAt              string                        `json:"createdAt,omitempty"`
	UpdatedAt              string                        `json:"updatedAt,omitempty"`
	Artifacts              []ManagedArtifactSyncArtifact `json:"artifacts,omitempty"`
}

// ManagedArtifactSyncList contains one page of synchronization records.
type ManagedArtifactSyncList struct {
	Syncs         []ManagedArtifactSync `json:"syncs"`
	NextPageToken string                `json:"nextPageToken,omitempty"`
}
