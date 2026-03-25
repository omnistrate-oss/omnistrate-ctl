package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

type resourceIndex struct {
	byKey          map[string]resourceMeta
	byID           map[string]string
	byName         map[string]string
	total          int
	terraformCount int
}

type resourceMeta struct {
	id          string
	isTerraform bool
}

type rawResourceFilter struct {
	key  string
	name string
	id   string
}

type resourceFilter struct {
	key string
	id  string
}

type terraformResourceEntry struct {
	key string
	id  string
}

func buildResourceIndex(ctx context.Context, token, serviceID string, instanceData *openapiclientfleet.ResourceInstance, includeNames bool) (*resourceIndex, error) {
	index := &resourceIndex{
		byKey:  make(map[string]resourceMeta),
		byID:   make(map[string]string),
		byName: make(map[string]string),
	}

	if instanceData == nil {
		return index, fmt.Errorf("instance data is nil")
	}

	needsVersionSet := false
	for _, summary := range instanceData.ResourceVersionSummaries {
		if summary.ResourceName == nil {
			if summary.ResourceId != nil {
				needsVersionSet = true
			}
			continue
		}

		resourceKey := *summary.ResourceName
		meta := index.byKey[resourceKey]

		if summary.ResourceId != nil && meta.id == "" {
			meta.id = *summary.ResourceId
			index.byID[meta.id] = resourceKey
		} else if summary.ResourceId == nil {
			needsVersionSet = true
		}

		if isTerraformResourceSummary(summary) {
			meta.isTerraform = true
		}

		index.byKey[resourceKey] = meta
	}

	if needsVersionSet || includeNames {
		versionSet, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, instanceData.ProductTierId, instanceData.TierVersion)
		if err != nil {
			return index, err
		}

		for _, resource := range versionSet.Resources {
			resourceKey := resourceKeyFromVersionSet(resource, index)
			meta := index.byKey[resourceKey]

			if meta.id == "" {
				meta.id = resource.Id
			}

			index.byID[resource.Id] = resourceKey
			index.byKey[resourceKey] = meta

			if includeNames {
				index.byName[resource.Name] = resourceKey
				if resource.UrlKey != nil && *resource.UrlKey != "" {
					index.byName[*resource.UrlKey] = resourceKey
				}
			}
		}
	}

	index.recount()
	return index, nil
}

func resourceKeyFromVersionSet(resource openapiclientv1.ResourceSummary, index *resourceIndex) string {
	if key, ok := index.byID[resource.Id]; ok {
		return key
	}
	if resource.UrlKey != nil && *resource.UrlKey != "" {
		return *resource.UrlKey
	}
	return resource.Name
}

func (index *resourceIndex) recount() {
	index.total = len(index.byKey)
	index.terraformCount = 0
	for _, meta := range index.byKey {
		if meta.isTerraform {
			index.terraformCount++
		}
	}
}

func resolveResourceFilter(raw rawResourceFilter, index *resourceIndex) (resourceFilter, error) {
	filter := resourceFilter{
		key: raw.key,
		id:  raw.id,
	}

	if raw.name != "" {
		keyFromName, ok := index.byName[raw.name]
		if !ok {
			return resourceFilter{}, fmt.Errorf("resource name '%s' not found", raw.name)
		}
		if filter.key != "" && filter.key != keyFromName {
			return resourceFilter{}, fmt.Errorf("resource-name '%s' does not match resource-key '%s'", raw.name, filter.key)
		}
		filter.key = keyFromName
	}

	if filter.key != "" {
		if meta, ok := index.byKey[filter.key]; ok && meta.id != "" {
			if filter.id != "" && filter.id != meta.id {
				return resourceFilter{}, fmt.Errorf("resource-key '%s' does not match resource-id '%s'", filter.key, filter.id)
			}
			if filter.id == "" {
				filter.id = meta.id
			}
		}
	}

	if filter.id != "" {
		if key, ok := index.byID[filter.id]; ok {
			if filter.key != "" && filter.key != key {
				return resourceFilter{}, fmt.Errorf("resource-id '%s' does not match resource-key '%s'", filter.id, filter.key)
			}
			filter.key = key
		}
	}

	return filter, nil
}

func (index *resourceIndex) needsTerraformData(filter resourceFilter) bool {
	if filter.key != "" {
		return index.isTerraformKey(filter.key)
	}
	if filter.id != "" {
		if key, ok := index.byID[filter.id]; ok {
			return index.isTerraformKey(key)
		}
		return strings.HasPrefix(filter.id, "tf-r-")
	}
	return index.terraformCount > 0
}

func (index *resourceIndex) terraformOnly(filter resourceFilter) bool {
	if filter.key != "" {
		return index.isTerraformKey(filter.key)
	}
	if filter.id != "" {
		if key, ok := index.byID[filter.id]; ok {
			return index.isTerraformKey(key)
		}
		return strings.HasPrefix(filter.id, "tf-r-")
	}
	return index.total > 0 && index.terraformCount == index.total
}

func (index *resourceIndex) isTerraformKey(resourceKey string) bool {
	meta, ok := index.byKey[resourceKey]
	return ok && meta.isTerraform
}

func (index *resourceIndex) resourceIDForKey(resourceKey string) (string, bool) {
	meta, ok := index.byKey[resourceKey]
	if !ok || meta.id == "" {
		return "", false
	}
	return meta.id, true
}

func listTerraformResources(instanceData *openapiclientfleet.ResourceInstance, index *resourceIndex, filter resourceFilter) []terraformResourceEntry {
	if instanceData == nil {
		return nil
	}

	var resources []terraformResourceEntry
	for _, summary := range instanceData.ResourceVersionSummaries {
		if !isTerraformResourceSummary(summary) {
			continue
		}

		if summary.ResourceName == nil {
			continue
		}

		resourceKey := *summary.ResourceName
		if filter.key != "" && filter.key != resourceKey {
			continue
		}

		resourceID := ""
		if summary.ResourceId != nil {
			resourceID = *summary.ResourceId
		} else if id, ok := index.resourceIDForKey(resourceKey); ok {
			resourceID = id
		}

		if filter.id != "" {
			if resourceID == "" || resourceID != filter.id {
				continue
			}
		}

		resources = append(resources, terraformResourceEntry{
			key: resourceKey,
			id:  resourceID,
		})
	}

	return resources
}

func isTerraformResourceSummary(summary openapiclientfleet.ResourceVersionSummary) bool {
	if summary.TerraformDeploymentConfiguration != nil {
		return true
	}

	if summary.ResourceId != nil && strings.HasPrefix(*summary.ResourceId, "tf-r-") {
		return true
	}

	return false
}
