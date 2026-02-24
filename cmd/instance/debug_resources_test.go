package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func TestResolveResourceFilter(t *testing.T) {
	index := &resourceIndex{
		byKey: map[string]resourceMeta{
			"db": {id: "tf-r-1", isTerraform: true},
		},
		byID: map[string]string{
			"tf-r-1": "db",
		},
		byName: map[string]string{
			"Database": "db",
		},
	}

	tests := []struct {
		name    string
		raw     rawResourceFilter
		want    resourceFilter
		wantErr bool
	}{
		{"by key", rawResourceFilter{key: "db"}, resourceFilter{key: "db", id: "tf-r-1"}, false},
		{"by name", rawResourceFilter{name: "Database"}, resourceFilter{key: "db", id: "tf-r-1"}, false},
		{"by id", rawResourceFilter{id: "tf-r-1"}, resourceFilter{key: "db", id: "tf-r-1"}, false},
		{"conflict key name", rawResourceFilter{key: "db", name: "Other"}, resourceFilter{}, true},
		{"conflict key id", rawResourceFilter{key: "db", id: "tf-r-2"}, resourceFilter{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveResourceFilter(tt.raw, index)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.key != tt.want.key || got.id != tt.want.id {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestResourceIndexTerraformSelection(t *testing.T) {
	index := &resourceIndex{
		byKey: map[string]resourceMeta{
			"db":    {id: "tf-r-1", isTerraform: true},
			"cache": {id: "res-2", isTerraform: false},
		},
		byID: map[string]string{
			"tf-r-1": "db",
			"res-2":  "cache",
		},
	}
	index.recount()

	if index.terraformOnly(resourceFilter{}) {
		t.Fatalf("expected terraformOnly=false for mixed resources")
	}
	if !index.terraformOnly(resourceFilter{key: "db"}) {
		t.Fatalf("expected terraformOnly=true for terraform key")
	}
	if index.terraformOnly(resourceFilter{key: "cache"}) {
		t.Fatalf("expected terraformOnly=false for non-terraform key")
	}
	if !index.terraformOnly(resourceFilter{id: "tf-r-1"}) {
		t.Fatalf("expected terraformOnly=true for terraform id")
	}
	if index.terraformOnly(resourceFilter{id: "res-2"}) {
		t.Fatalf("expected terraformOnly=false for non-terraform id")
	}
	if !index.terraformOnly(resourceFilter{id: "tf-r-unknown"}) {
		t.Fatalf("expected terraformOnly=true for tf-r- prefix")
	}
}

func TestListTerraformResources(t *testing.T) {
	terraformConfig := &openapiclientfleet.TerraformDeploymentConfiguration{}
	instanceData := &openapiclientfleet.ResourceInstance{
		ResourceVersionSummaries: []openapiclientfleet.ResourceVersionSummary{
			{
				ResourceName:                     strPtr("db"),
				ResourceId:                       strPtr("tf-r-1"),
				TerraformDeploymentConfiguration: terraformConfig,
			},
			{
				ResourceName: strPtr("cache"),
				ResourceId:   strPtr("res-2"),
			},
		},
	}

	index := &resourceIndex{
		byKey: map[string]resourceMeta{
			"db": {id: "tf-r-1", isTerraform: true},
		},
	}

	results := listTerraformResources(instanceData, index, resourceFilter{})
	if len(results) != 1 {
		t.Fatalf("expected 1 terraform resource, got %d", len(results))
	}
	if results[0].key != "db" || results[0].id != "tf-r-1" {
		t.Fatalf("unexpected terraform resource: %+v", results[0])
	}
}

func strPtr(val string) *string {
	return &val
}
