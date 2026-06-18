package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAmenityYAMLUnmarshalDependsOn(t *testing.T) {
	input := []byte(`customAmenities:
  - name: dependent
    type: Helm
    dependsOn:
      - namespace
      - crds
`)

	var template DeploymentCellTemplate
	if err := yaml.Unmarshal(input, &template); err != nil {
		t.Fatalf("failed to unmarshal template: %v", err)
	}

	if len(template.CustomAmenities) != 1 {
		t.Fatalf("expected 1 custom amenity, got %d", len(template.CustomAmenities))
	}

	got := template.CustomAmenities[0].DependsOn
	if len(got) != 2 || got[0] != "namespace" || got[1] != "crds" {
		t.Fatalf("unexpected dependsOn: %#v", got)
	}
}

func TestProcessManifestAmenities_FileReference(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test manifest file
	testManifest := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: nginx
    image: nginx:latest`

	manifestFile := filepath.Join(tmpDir, "test-manifest.yaml")
	if err := os.WriteFile(manifestFile, []byte(testManifest), 0600); err != nil {
		t.Fatalf("failed to create test manifest file: %v", err)
	}

	// Create an amenity with a file reference
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name:      "test-secrets",
			Type:      &manifestType,
			DependsOn: []string{"namespace"},
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "test-manifest.yaml"},
				},
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	// Verify the result
	if len(result) != 1 {
		t.Fatalf("expected 1 amenity, got %d", len(result))
	}
	if len(result[0].DependsOn) != 1 || result[0].DependsOn[0] != "namespace" {
		t.Fatalf("unexpected dependsOn: %#v", result[0].DependsOn)
	}

	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}

	def, ok := manifests[0]["def"].(map[string]interface{})
	if !ok {
		t.Fatalf("def property is not a map[string]interface{}")
	}

	// Verify parsed YAML content
	if def["apiVersion"] != "v1" {
		t.Errorf("expected apiVersion 'v1', got %v", def["apiVersion"])
	}
	if def["kind"] != "Pod" {
		t.Errorf("expected kind 'Pod', got %v", def["kind"])
	}

	// Verify file reference is removed
	if _, ok := manifests[0]["file"]; ok {
		t.Error("file property should not be present in processed manifest")
	}
}

func TestProcessManifestAmenitiesPreservesDisable(t *testing.T) {
	tmpDir := t.TempDir()
	testManifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config`

	manifestFile := filepath.Join(tmpDir, "test-manifest.yaml")
	if err := os.WriteFile(manifestFile, []byte(testManifest), 0600); err != nil {
		t.Fatalf("failed to create test manifest file: %v", err)
	}

	manifestType := AmenityTypeKubernetesManifest
	disable := `$sys.deploymentCell.isImported`
	amenities := []Amenity{
		{
			Name:    "test-config",
			Type:    &manifestType,
			Disable: &disable,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "test-manifest.yaml"},
				},
			},
		},
	}

	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 amenity, got %d", len(result))
	}
	if result[0].Disable == nil || *result[0].Disable != disable {
		t.Fatalf("unexpected disable: %#v", result[0].Disable)
	}
}

func TestProcessManifestAmenities_InlineDefinition(t *testing.T) {
	// Create an amenity with an inline definition as a map
	manifestType := AmenityTypeKubernetesManifest
	inlineDef := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "test-config",
		},
	}

	amenities := []Amenity{
		{
			Name: "test-inline",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"def": inlineDef},
				},
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, "/tmp")
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	// Verify the result - inline def should be preserved
	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	def, ok := manifests[0]["def"].(map[string]interface{})
	if !ok {
		t.Fatalf("def property is not a map[string]interface{}")
	}

	if def["apiVersion"] != "v1" {
		t.Errorf("expected apiVersion 'v1', got %v", def["apiVersion"])
	}
	if def["kind"] != "ConfigMap" {
		t.Errorf("expected kind 'ConfigMap', got %v", def["kind"])
	}
}

func TestProcessManifestAmenities_MixedEntries(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create test manifest files
	manifest1 := `apiVersion: v1
kind: Secret
metadata:
  name: secret1`

	manifest2 := `apiVersion: v1
kind: Secret
metadata:
  name: secret2`

	if err := os.WriteFile(filepath.Join(tmpDir, "secret1.yaml"), []byte(manifest1), 0600); err != nil {
		t.Fatalf("failed to create secret1.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "secret2.yaml"), []byte(manifest2), 0600); err != nil {
		t.Fatalf("failed to create secret2.yaml: %v", err)
	}

	// Inline definition as a map
	inlineDef := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name": "inline-pod",
		},
	}

	// Create an amenity with mixed entries
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "mixed-secrets",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "./secret1.yaml"},
					map[string]interface{}{"file": "./secret2.yaml"},
					map[string]interface{}{"def": inlineDef},
				},
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	// Verify the result
	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	if len(manifests) != 3 {
		t.Fatalf("expected 3 manifests, got %d", len(manifests))
	}

	// Check all have def with correct kind
	expectedKinds := []string{"Secret", "Secret", "Pod"}
	for i, expectedKind := range expectedKinds {
		def, ok := manifests[i]["def"].(map[string]interface{})
		if !ok {
			t.Fatalf("manifest %d: def property is not a map[string]interface{}", i)
		}
		if def["kind"] != expectedKind {
			t.Errorf("manifest %d: expected kind '%s', got '%v'", i, expectedKind, def["kind"])
		}
	}
}

func TestProcessManifestAmenities_NonManifestType(t *testing.T) {
	// Create an amenity with a different type - should be passed through unchanged
	otherType := "helm"
	amenities := []Amenity{
		{
			Name: "helm-chart",
			Type: &otherType,
			Properties: map[string]interface{}{
				"chart": "nginx",
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, "/tmp")
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	// Verify the result - should be unchanged
	if len(result) != 1 {
		t.Fatalf("expected 1 amenity, got %d", len(result))
	}

	if result[0].Properties["chart"] != "nginx" {
		t.Error("helm amenity properties should be unchanged")
	}
}

func TestProcessManifestAmenities_FileNotFound(t *testing.T) {
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "missing-file",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "nonexistent.yaml"},
				},
			},
		},
	}

	// Process the amenities - should fail
	_, err := ProcessManifestAmenities(amenities, "/tmp")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestProcessManifestAmenities_InvalidEntry(t *testing.T) {
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "invalid-entry",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{}, // No file or def
				},
			},
		},
	}

	// Process the amenities - should fail
	_, err := ProcessManifestAmenities(amenities, "/tmp")
	if err == nil {
		t.Fatal("expected error for invalid entry, got nil")
	}
}

func TestProcessManifestAmenities_BothFileAndDef(t *testing.T) {
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "both-file-and-def",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{
						"file": "some-file.yaml",
						"def": map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Pod",
						},
					},
				},
			},
		},
	}

	// Process the amenities - should fail because both file and def are specified
	_, err := ProcessManifestAmenities(amenities, "/tmp")
	if err == nil {
		t.Fatal("expected error when both file and def are specified, got nil")
	}
}

func TestProcessManifestAmenities_EmptyManifests(t *testing.T) {
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "empty-manifests",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{},
			},
		},
	}

	// Process the amenities - should fail because manifests array is empty
	_, err := ProcessManifestAmenities(amenities, "/tmp")
	if err == nil {
		t.Fatal("expected error for empty manifests array, got nil")
	}
}

func TestProcessManifestAmenities_InvalidYAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create an invalid YAML file
	invalidYAML := `this is not valid yaml: [unclosed bracket`
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("failed to create invalid.yaml: %v", err)
	}

	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "invalid-yaml",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "invalid.yaml"},
				},
			},
		},
	}

	// Process the amenities - should fail because YAML is invalid
	_, err := ProcessManifestAmenities(amenities, tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestProcessManifestAmenities_OnlyEmptyDocuments(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a YAML file that contains only document separators (no actual content)
	emptyDocsYAML := "---\n---\n---\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "empty-docs.yaml"), []byte(emptyDocsYAML), 0600); err != nil {
		t.Fatalf("failed to create empty-docs.yaml: %v", err)
	}

	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "empty-docs",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "empty-docs.yaml"},
				},
			},
		},
	}

	// Process the amenities - should fail because file contains no valid YAML documents
	_, err := ProcessManifestAmenities(amenities, tmpDir)
	if err == nil {
		t.Fatal("expected error for file with only empty YAML documents, got nil")
	}

	// Verify the error message is descriptive
	expectedSubstring := "contains no valid YAML documents"
	if !strings.Contains(err.Error(), expectedSubstring) {
		t.Errorf("expected error to contain %q, got: %v", expectedSubstring, err)
	}
}

func TestProcessManifestAmenities_MultiDocumentYAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a multi-document YAML file with three documents
	multiDocYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-1
  namespace: default
data:
  key1: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-2
  namespace: default
data:
  key2: value2
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-1
  namespace: default
type: Opaque
stringData:
  password: s3cr3t`

	manifestFile := filepath.Join(tmpDir, "multi-doc.yaml")
	if err := os.WriteFile(manifestFile, []byte(multiDocYAML), 0600); err != nil {
		t.Fatalf("failed to create multi-doc.yaml: %v", err)
	}

	// Create an amenity with a file reference to the multi-document file
	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "multi-doc-manifests",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "multi-doc.yaml"},
				},
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	// Verify the result
	if len(result) != 1 {
		t.Fatalf("expected 1 amenity, got %d", len(result))
	}

	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	// Should have 3 manifests from the 3 documents
	if len(manifests) != 3 {
		t.Fatalf("expected 3 manifests from multi-document file, got %d", len(manifests))
	}

	// Check each document was parsed correctly
	expectedKinds := []string{"ConfigMap", "ConfigMap", "Secret"}
	expectedNames := []string{"config-1", "config-2", "secret-1"}
	for i, expectedKind := range expectedKinds {
		def, ok := manifests[i]["def"].(map[string]interface{})
		if !ok {
			t.Fatalf("manifest %d: def property is not a map[string]interface{}", i)
		}
		if def["kind"] != expectedKind {
			t.Errorf("manifest %d: expected kind '%s', got '%v'", i, expectedKind, def["kind"])
		}
		metadata, ok := def["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("manifest %d: metadata is not a map[string]interface{}", i)
		}
		if metadata["name"] != expectedNames[i] {
			t.Errorf("manifest %d: expected name '%s', got '%v'", i, expectedNames[i], metadata["name"])
		}
	}
}

func TestProcessManifestAmenities_MultiDocumentWithTrailingSeparator(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Multi-document YAML with a trailing '---' (empty final document)
	multiDocYAML := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config-1
data:
  key: value
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-2
data:
  key: value
---
`

	manifestFile := filepath.Join(tmpDir, "trailing-sep.yaml")
	if err := os.WriteFile(manifestFile, []byte(multiDocYAML), 0600); err != nil {
		t.Fatalf("failed to create trailing-sep.yaml: %v", err)
	}

	manifestType := AmenityTypeKubernetesManifest
	amenities := []Amenity{
		{
			Name: "trailing-separator",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "trailing-sep.yaml"},
				},
			},
		},
	}

	// Process the amenities - should succeed, ignoring the empty trailing document
	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	// Should only have 2 valid documents (trailing empty document is skipped)
	if len(manifests) != 2 {
		t.Fatalf("expected 2 manifests (empty trailing doc skipped), got %d", len(manifests))
	}
}

func TestProcessManifestAmenities_MultiDocumentMixedWithInline(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Multi-document file
	multiDocYAML := `apiVersion: v1
kind: Namespace
metadata:
  name: my-ns
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
  namespace: my-ns
data:
  key: value`

	manifestFile := filepath.Join(tmpDir, "multi.yaml")
	if err := os.WriteFile(manifestFile, []byte(multiDocYAML), 0600); err != nil {
		t.Fatalf("failed to create multi.yaml: %v", err)
	}

	// Create an amenity with both a multi-doc file and an inline def
	manifestType := AmenityTypeKubernetesManifest
	inlineDef := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name": "my-secret",
		},
	}
	amenities := []Amenity{
		{
			Name: "mixed-multi-doc",
			Type: &manifestType,
			Properties: map[string]interface{}{
				"manifests": []interface{}{
					map[string]interface{}{"file": "multi.yaml"},
					map[string]interface{}{"def": inlineDef},
				},
			},
		},
	}

	// Process the amenities
	result, err := ProcessManifestAmenities(amenities, tmpDir)
	if err != nil {
		t.Fatalf("ProcessManifestAmenities failed: %v", err)
	}

	props := result[0].Properties
	manifests, ok := props["manifests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("manifests property is not []map[string]interface{}")
	}

	// Should have 3 manifests: 2 from the multi-doc file + 1 inline
	if len(manifests) != 3 {
		t.Fatalf("expected 3 manifests (2 from file + 1 inline), got %d", len(manifests))
	}

	// Verify order: Namespace, ConfigMap (from file), then Secret (inline)
	expectedKinds := []string{"Namespace", "ConfigMap", "Secret"}
	for i, expectedKind := range expectedKinds {
		def, ok := manifests[i]["def"].(map[string]interface{})
		if !ok {
			t.Fatalf("manifest %d: def property is not a map[string]interface{}", i)
		}
		if def["kind"] != expectedKind {
			t.Errorf("manifest %d: expected kind '%s', got '%v'", i, expectedKind, def["kind"])
		}
	}
}

func TestManifestEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		entry   ManifestEntry
		wantErr bool
	}{
		{
			name:    "valid file entry",
			entry:   ManifestEntry{File: "test.yaml"},
			wantErr: false,
		},
		{
			name: "valid def entry",
			entry: ManifestEntry{Def: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
			}},
			wantErr: false,
		},
		{
			name:    "empty entry",
			entry:   ManifestEntry{},
			wantErr: true,
		},
		{
			name: "both file and def",
			entry: ManifestEntry{
				File: "test.yaml",
				Def: map[string]interface{}{
					"apiVersion": "v1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate(0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestEntry.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManifestProperties_Validate(t *testing.T) {
	tests := []struct {
		name    string
		props   ManifestProperties
		wantErr bool
	}{
		{
			name: "valid single entry",
			props: ManifestProperties{
				Manifests: []ManifestEntry{{File: "test.yaml"}},
			},
			wantErr: false,
		},
		{
			name: "valid multiple entries",
			props: ManifestProperties{
				Manifests: []ManifestEntry{
					{File: "test1.yaml"},
					{Def: map[string]interface{}{"apiVersion": "v1"}},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty manifests",
			props:   ManifestProperties{Manifests: []ManifestEntry{}},
			wantErr: true,
		},
		{
			name:    "nil manifests",
			props:   ManifestProperties{},
			wantErr: true,
		},
		{
			name: "invalid entry in list",
			props: ManifestProperties{
				Manifests: []ManifestEntry{
					{File: "valid.yaml"},
					{}, // invalid
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.props.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestProperties.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
