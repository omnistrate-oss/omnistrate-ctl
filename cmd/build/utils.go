package build

import (
	"strings"
)

// DetectSpecType analyzes YAML content to determine if it contains service plan specifications
// Returns ServicePlanSpecType if plan-specific keys are found, otherwise DockerComposeSpecType
func DetectSpecType(yamlContent map[string]interface{}) string {
	// Improved: Recursively check for plan spec keys at any level
	planKeyGroups := [][]string{
		{"helm", "helmChart", "helmChartConfiguration"},
		{"operator", "operatorCRDConfiguration"},
		{"terraform", "terraformConfigurations"},
		{"kustomize", "kustomizeConfiguration"},
	}

	// Check if any plan-specific keys are found
	for _, keys := range planKeyGroups {
		if ContainsAnyKey(yamlContent, keys) {
			return ServicePlanSpecType
		}
	}

	return DockerComposeSpecType
}

// ContainsOmnistrateKey recursively searches for any x-omnistrate key in a map
func ContainsOmnistrateKey(m map[string]interface{}) bool {
	for k, v := range m {
		// Check for any x-omnistrate key
		if strings.HasPrefix(k, "x-omnistrate-") {
			return true
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]interface{}); ok {
			if ContainsOmnistrateKey(sub) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]interface{}); ok {
					if ContainsOmnistrateKey(subm) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ContainsAnyKey recursively searches for any key in keys in a map
func ContainsAnyKey(m map[string]interface{}, keys []string) bool {
	for k, v := range m {
		for _, key := range keys {
			if k == key {
				return true
			}
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]interface{}); ok {
			if ContainsAnyKey(sub, keys) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]interface{}); ok {
					if ContainsAnyKey(subm, keys) {
						return true
					}
				}
			}
		}
	}
	return false
}
