package dataaccess

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

const (
	customerMetricsFeatureKey = "METRICS"
	internalMetricsFeatureKey = "METRICS#INTERNAL"
)

type DashboardService struct{}

type DashboardCatalog struct {
	InstanceID          string                 `json:"instanceId,omitempty"`
	PreferredFeatureKey string                 `json:"preferredFeatureKey,omitempty"`
	Features            []DashboardFeatureInfo `json:"features,omitempty"`
}

type DashboardFeatureInfo struct {
	Key                  string                `json:"key,omitempty"`
	Label                string                `json:"label,omitempty"`
	GrafanaEndpoint      string                `json:"grafanaEndpoint,omitempty"`
	GrafanaUIUsername    string                `json:"grafanaUiUsername,omitempty"`
	GrafanaUIPassword    string                `json:"grafanaUiPassword,omitempty"`
	GrafanaUILoginScope  string                `json:"grafanaUiLoginScope,omitempty"`
	ServiceAccountName   string                `json:"serviceAccountUsername,omitempty"`
	ServiceAccountToken  string                `json:"serviceAccountToken,omitempty"`
	Dashboards           []DashboardRef        `json:"dashboards,omitempty"`
	DashboardDefinitions []DashboardDefinition `json:"dashboardDefinitions,omitempty"`
}

type DashboardInfo struct {
	InstanceID           string                `json:"instanceId,omitempty"`
	MetricsFeatureKey    string                `json:"metricsFeatureKey,omitempty"`
	MetricsFeatureLabel  string                `json:"metricsFeatureLabel,omitempty"`
	GrafanaEndpoint      string                `json:"grafanaEndpoint,omitempty"`
	GrafanaLoginUsername string                `json:"grafanaLoginUsername,omitempty"`
	GrafanaLoginPassword string                `json:"grafanaLoginPassword,omitempty"`
	GrafanaLoginScope    string                `json:"grafanaLoginScope,omitempty"`
	ServiceAccountName   string                `json:"serviceAccountName,omitempty"`
	ServiceAccountToken  string                `json:"serviceAccountToken,omitempty"`
	Dashboards           []DashboardRef        `json:"dashboards,omitempty"`
	DashboardDefinitions []DashboardDefinition `json:"dashboardDefinitions,omitempty"`
}

type DashboardRef struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type DashboardDefinition struct {
	Source string `json:"source,omitempty"`
	Name   string `json:"name,omitempty"`
	Title  string `json:"title,omitempty"`
}

type metricsFeatureConfig struct {
	Enabled                bool                                       `json:"enabled"`
	GrafanaEndpoint        string                                     `json:"grafanaEndpoint,omitempty"`
	ServiceAccountUsername string                                     `json:"serviceAccountUsername,omitempty"`
	ServiceAccountPassword string                                     `json:"serviceAccountPassword,omitempty"`
	InstanceOrgID          string                                     `json:"instanceOrgId,omitempty"`
	InstanceOrgPassword    string                                     `json:"instanceOrgPassword,omitempty"`
	ServiceProviderOrgID   string                                     `json:"serviceProviderOrgId,omitempty"`
	ServiceProviderOrgPass string                                     `json:"serviceProviderOrgPassword,omitempty"`
	Dashboards             map[string]metricsFeatureDashboard         `json:"dashboards,omitempty"`
	AdditionalMetrics      map[string]metricsFeatureAdditionalMetrics `json:"additionalMetrics,omitempty"`
}

type metricsFeatureDashboard struct {
	Description   string `json:"description,omitempty"`
	DashboardLink string `json:"dashboardLink,omitempty"`
}

type metricsFeatureAdditionalMetrics struct {
	Dashboards map[string]metricsFeatureAdditionalDashboard `json:"dashboards,omitempty"`
}

type metricsFeatureAdditionalDashboard struct {
	Title string `json:"title,omitempty"`
}

type resolvedMetricsFeature struct {
	key     string
	feature metricsFeatureConfig
}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

func (d *DashboardService) IsMetricsEnabled(instance *openapiclientfleet.ResourceInstance) bool {
	_, err := d.resolveEnabledMetricsFeatures(instance)
	return err == nil
}

func (d *DashboardService) GetDashboardCatalog(instance *openapiclientfleet.ResourceInstance) (*DashboardCatalog, error) {
	resolvedFeatures, err := d.resolveEnabledMetricsFeatures(instance)
	if err != nil {
		return nil, err
	}

	catalog := &DashboardCatalog{}
	if instance != nil && instance.ConsumptionResourceInstanceResult.Id != nil {
		catalog.InstanceID = *instance.ConsumptionResourceInstanceResult.Id
	}
	if len(resolvedFeatures) > 0 {
		catalog.PreferredFeatureKey = resolvedFeatures[0].key
	}

	catalog.Features = make([]DashboardFeatureInfo, 0, len(resolvedFeatures))
	for _, resolvedFeature := range resolvedFeatures {
		featureInfo, buildErr := buildDashboardFeatureInfo(resolvedFeature.key, resolvedFeature.feature)
		if buildErr != nil {
			return nil, buildErr
		}
		catalog.Features = append(catalog.Features, featureInfo)
	}

	return catalog, nil
}

func (d *DashboardService) GetDashboardInfo(instance *openapiclientfleet.ResourceInstance) (*DashboardInfo, error) {
	catalog, err := d.GetDashboardCatalog(instance)
	if err != nil {
		return nil, err
	}

	feature := catalog.PreferredFeature()
	if feature == nil {
		return nil, fmt.Errorf("metrics are not enabled for this instance")
	}

	return &DashboardInfo{
		InstanceID:           catalog.InstanceID,
		MetricsFeatureKey:    feature.Key,
		MetricsFeatureLabel:  feature.Label,
		GrafanaEndpoint:      feature.GrafanaEndpoint,
		GrafanaLoginUsername: feature.GrafanaUIUsername,
		GrafanaLoginPassword: feature.GrafanaUIPassword,
		GrafanaLoginScope:    feature.GrafanaUILoginScope,
		ServiceAccountName:   feature.ServiceAccountName,
		ServiceAccountToken:  feature.ServiceAccountToken,
		Dashboards:           append([]DashboardRef(nil), feature.Dashboards...),
		DashboardDefinitions: append([]DashboardDefinition(nil), feature.DashboardDefinitions...),
	}, nil
}

func (c *DashboardCatalog) PreferredFeature() *DashboardFeatureInfo {
	if c == nil || len(c.Features) == 0 {
		return nil
	}

	if c.PreferredFeatureKey != "" {
		for index := range c.Features {
			if c.Features[index].Key == c.PreferredFeatureKey {
				return &c.Features[index]
			}
		}
	}

	return &c.Features[0]
}

func (f *DashboardFeatureInfo) HasAccessCredentials() bool {
	if f == nil {
		return false
	}

	hasUILogin := f.GrafanaUIUsername != "" && f.GrafanaUIPassword != ""
	hasServiceAccount := f.ServiceAccountName != "" && f.ServiceAccountToken != ""
	return hasUILogin || hasServiceAccount
}

func (d *DashboardService) resolveEnabledMetricsFeatures(instance *openapiclientfleet.ResourceInstance) ([]resolvedMetricsFeature, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	features := instance.ConsumptionResourceInstanceResult.ProductTierFeatures
	if features == nil {
		return nil, fmt.Errorf("metrics are not enabled for this instance")
	}

	resolvedFeatures := make([]resolvedMetricsFeature, 0, 2)
	var disabledFeatureKey string

	for _, featureKey := range []string{customerMetricsFeatureKey, internalMetricsFeatureKey} {
		rawFeature, ok := features[featureKey]
		if !ok {
			continue
		}

		feature, err := decodeMetricsFeature(rawFeature)
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s feature: %w", featureKey, err)
		}

		if feature.Enabled {
			resolvedFeatures = append(resolvedFeatures, resolvedMetricsFeature{key: featureKey, feature: feature})
			continue
		}

		if disabledFeatureKey == "" {
			disabledFeatureKey = featureKey
		}
	}

	if len(resolvedFeatures) > 0 {
		return resolvedFeatures, nil
	}

	if disabledFeatureKey != "" {
		return nil, fmt.Errorf("%s is disabled for this instance", disabledFeatureKey)
	}

	return nil, fmt.Errorf("metrics are not enabled for this instance")
}

func buildDashboardFeatureInfo(featureKey string, feature metricsFeatureConfig) (DashboardFeatureInfo, error) {
	info := DashboardFeatureInfo{
		Key:                 featureKey,
		Label:               metricsFeatureLabel(featureKey),
		GrafanaEndpoint:     feature.GrafanaEndpoint,
		ServiceAccountName:  feature.ServiceAccountUsername,
		ServiceAccountToken: feature.ServiceAccountPassword,
	}

	switch {
	case feature.InstanceOrgID != "" || feature.InstanceOrgPassword != "":
		info.GrafanaUIUsername = feature.InstanceOrgID
		info.GrafanaUIPassword = feature.InstanceOrgPassword
		info.GrafanaUILoginScope = "customer"
	case feature.ServiceProviderOrgID != "" || feature.ServiceProviderOrgPass != "":
		info.GrafanaUIUsername = feature.ServiceProviderOrgID
		info.GrafanaUIPassword = feature.ServiceProviderOrgPass
		info.GrafanaUILoginScope = "provider"
	}

	if len(feature.Dashboards) > 0 {
		dashboardNames := make([]string, 0, len(feature.Dashboards))
		for name := range feature.Dashboards {
			dashboardNames = append(dashboardNames, name)
		}
		sort.Strings(dashboardNames)

		info.Dashboards = make([]DashboardRef, 0, len(dashboardNames))
		for _, name := range dashboardNames {
			dashboard := feature.Dashboards[name]
			info.Dashboards = append(info.Dashboards, DashboardRef{
				Name:        name,
				Description: dashboard.Description,
				URL:         dashboard.DashboardLink,
			})
			if info.GrafanaEndpoint == "" && dashboard.DashboardLink != "" {
				info.GrafanaEndpoint = dashboardBaseURL(dashboard.DashboardLink)
			}
		}
	}

	for source, additionalMetrics := range feature.AdditionalMetrics {
		for name, dashboard := range additionalMetrics.Dashboards {
			info.DashboardDefinitions = append(info.DashboardDefinitions, DashboardDefinition{
				Source: source,
				Name:   name,
				Title:  dashboard.Title,
			})
		}
	}
	sort.Slice(info.DashboardDefinitions, func(i, j int) bool {
		if info.DashboardDefinitions[i].Source == info.DashboardDefinitions[j].Source {
			return info.DashboardDefinitions[i].Name < info.DashboardDefinitions[j].Name
		}
		return info.DashboardDefinitions[i].Source < info.DashboardDefinitions[j].Source
	})

	if info.GrafanaEndpoint == "" {
		return DashboardFeatureInfo{}, fmt.Errorf("%s is enabled for this instance, but the Grafana endpoint is not available in the instance metadata", featureKey)
	}

	if !info.HasAccessCredentials() {
		return DashboardFeatureInfo{}, fmt.Errorf("%s is enabled for this instance, but Grafana access credentials are not available in the instance metadata", featureKey)
	}

	return info, nil
}

func metricsFeatureLabel(featureKey string) string {
	switch featureKey {
	case customerMetricsFeatureKey:
		return "Customer"
	case internalMetricsFeatureKey:
		return "Internal"
	default:
		return featureKey
	}
}

func decodeMetricsFeature(rawFeature any) (metricsFeatureConfig, error) {
	encoded, err := json.Marshal(rawFeature)
	if err != nil {
		return metricsFeatureConfig{}, err
	}

	var feature metricsFeatureConfig
	if err = json.Unmarshal(encoded, &feature); err != nil {
		return metricsFeatureConfig{}, err
	}

	return feature, nil
}

func dashboardBaseURL(link string) string {
	parsed, err := url.Parse(link)
	if err != nil {
		return ""
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}
