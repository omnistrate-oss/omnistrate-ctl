package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// AmenityTypeManifest is the type identifier for Kubernetes manifest amenities
	AmenityTypeManifest = "manifest"
)

// ManifestEntry represents a single manifest entry which can be either a file reference or an inline definition
type ManifestEntry struct {
	File string                 `json:"file,omitempty" yaml:"file,omitempty"`
	Def  map[string]interface{} `json:"def,omitempty" yaml:"def,omitempty"`
}

// Validate checks that the manifest entry has exactly one of file or def set
func (e ManifestEntry) Validate(index int) error {
	hasFile := e.File != ""
	hasDef := len(e.Def) > 0

	if !hasFile && !hasDef {
		return fmt.Errorf("manifest entry %d: must have either 'file' or 'def' property", index)
	}
	if hasFile && hasDef {
		return fmt.Errorf("manifest entry %d: cannot have both 'file' and 'def' properties", index)
	}
	return nil
}

// ManifestProperties represents the properties structure for a manifest amenity
type ManifestProperties struct {
	Manifests []ManifestEntry `json:"manifests,omitempty" yaml:"manifests,omitempty"`
}

// Validate checks that the manifest properties are valid
func (p ManifestProperties) Validate() error {
	if len(p.Manifests) == 0 {
		return fmt.Errorf("manifests array cannot be empty")
	}
	for i, entry := range p.Manifests {
		if err := entry.Validate(i); err != nil {
			return err
		}
	}
	return nil
}

// ProcessManifestAmenities processes all amenities and converts file-based manifest references
// to inline definitions. The baseDir is used to resolve relative file paths.
func ProcessManifestAmenities(amenities []Amenity, baseDir string) ([]Amenity, error) {
	result := make([]Amenity, 0, len(amenities))

	for _, amenity := range amenities {
		processed, err := processAmenity(amenity, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to process amenity '%s': %w", amenity.Name, err)
		}
		result = append(result, processed)
	}

	return result, nil
}

// processAmenity processes a single amenity and converts file references to inline definitions
// if it's a manifest type amenity
func processAmenity(amenity Amenity, baseDir string) (Amenity, error) {
	// Only process manifest type amenities
	if amenity.Type == nil || *amenity.Type != AmenityTypeManifest {
		return amenity, nil
	}

	// Check if properties exist
	if amenity.Properties == nil {
		return amenity, nil
	}

	// Check if manifests key exists
	if _, ok := amenity.Properties["manifests"]; !ok {
		return amenity, nil
	}

	// Convert properties map to ManifestProperties struct for validation
	manifestProps, err := parseManifestProperties(amenity.Properties)
	if err != nil {
		return amenity, fmt.Errorf("invalid manifest properties: %w", err)
	}

	// Validate the manifest properties
	if err := manifestProps.Validate(); err != nil {
		return amenity, err
	}

	// Process each manifest entry and convert file references to inline definitions
	processedEntries := make([]ManifestEntry, 0, len(manifestProps.Manifests))
	for i, entry := range manifestProps.Manifests {
		processed, err := processManifestEntry(entry, baseDir, i)
		if err != nil {
			return amenity, err
		}
		processedEntries = append(processedEntries, processed)
	}

	// Create a new amenity with processed properties
	newProperties := make(map[string]interface{})
	for k, v := range amenity.Properties {
		if k != "manifests" {
			newProperties[k] = v
		}
	}

	// Convert processed entries back to []map[string]interface{} for API compatibility
	processedManifests := make([]map[string]interface{}, 0, len(processedEntries))
	for _, entry := range processedEntries {
		processedManifests = append(processedManifests, map[string]interface{}{"def": entry.Def})
	}
	newProperties["manifests"] = processedManifests

	return Amenity{
		Name:        amenity.Name,
		Description: amenity.Description,
		Type:        amenity.Type,
		Properties:  newProperties,
	}, nil
}

// parseManifestProperties converts the raw properties map to a ManifestProperties struct
func parseManifestProperties(properties map[string]interface{}) (ManifestProperties, error) {
	// Marshal to JSON then unmarshal to struct for type-safe conversion
	jsonBytes, err := json.Marshal(properties)
	if err != nil {
		return ManifestProperties{}, fmt.Errorf("failed to marshal properties: %w", err)
	}

	var manifestProps ManifestProperties
	if err := json.Unmarshal(jsonBytes, &manifestProps); err != nil {
		return ManifestProperties{}, fmt.Errorf("failed to parse manifest properties: %w", err)
	}

	return manifestProps, nil
}

// processManifestEntry processes a single manifest entry, reading file content if necessary
func processManifestEntry(entry ManifestEntry, baseDir string, index int) (ManifestEntry, error) {
	// If it already has a def, return as-is
	if len(entry.Def) > 0 {
		return ManifestEntry{Def: entry.Def}, nil
	}

	// Read the file and parse YAML to map
	filePath := entry.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(baseDir, filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return ManifestEntry{}, fmt.Errorf("manifest entry %d: failed to read file '%s': %w", index, filePath, err)
	}

	// Parse YAML content into a map
	var defMap map[string]interface{}
	if err := yaml.Unmarshal(content, &defMap); err != nil {
		return ManifestEntry{}, fmt.Errorf("manifest entry %d: failed to parse YAML from file '%s': %w", index, filePath, err)
	}

	return ManifestEntry{Def: defMap}, nil
}
