package dataaccess

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlugifyHeader pins the anchor generation against real heading/anchor pairs
// taken from the live compose spec and plan spec pages. These are the headings the
// naive "lowercase, spaces to hyphens, URL escape" approach got wrong.
func TestSlugifyHeader(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		// Code spans in headings must not survive into the anchor
		{"`Compute schema`", "compute-schema"},
		{"`API parameters schema`", "api-parameters-schema"},
		// Dotted extension paths lose the dots entirely
		{"x-omnistrate-capabilities.sidecars", "x-omnistrate-capabilitiessidecars"},
		{"x-omnistrate-service-plan.tenancyType", "x-omnistrate-service-plantenancytype"},
		{
			"x-omnistrate-compute.instanceTypes.configurationOverrides.acceleratorConfiguration",
			"x-omnistrate-computeinstancetypesconfigurationoverridesacceleratorconfiguration",
		},
		// Underscores are word characters and are preserved
		{"x-omnistrate-service-plan.features.CUSTOM_NETWORKS", "x-omnistrate-service-planfeaturescustom_networks"},
		{"cap_add / cap_drop", "cap_add-cap_drop"},
		{"container_name / hostname", "container_name-hostname"},
		// Parenthesised notes collapse into hyphens
		{"x-omnistrate-my-account (deprecated)", "x-omnistrate-my-account-deprecated"},
		{"customerMetrics schema (advanced metrics and dashboards)", "customermetrics-schema-advanced-metrics-and-dashboards"},
		{
			"Omnistrate Service Specification Overview (Helm, Operators, Kustomize, Terraform, OpenTofu)",
			"omnistrate-service-specification-overview-helm-operators-kustomize-terraform-opentofu",
		},
		// Colons are dropped, and the whitespace around them collapses to one hyphen
		{"volumes.driver: sharedFileSystem", "volumesdriver-sharedfilesystem"},
		{"volumes.driver: blob", "volumesdriver-blob"},
		// Ordinary headings
		{"L4 Load Balancer Configuration schema", "l4-load-balancer-configuration-schema"},
		{"Basic Structure", "basic-structure"},
		{"x-omnistrate-compute", "x-omnistrate-compute"},
	}

	for _, test := range tests {
		t.Run(test.header, func(t *testing.T) {
			assert.Equal(t, test.want, slugifyHeader(test.header))
		})
	}
}

func TestCleanHeaderText(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"`Compute schema`", "Compute schema"},
		{"`API parameters schema`", "API parameters schema"},
		{"**Bold heading**", "Bold heading"},
		{"Root schema", "Root schema"},
		// Underscores carry meaning in tag names and must be left alone
		{"cap_add / cap_drop", "cap_add / cap_drop"},
		{"x-omnistrate-service-plan.features.CUSTOM_NETWORKS", "x-omnistrate-service-plan.features.CUSTOM_NETWORKS"},
	}

	for _, test := range tests {
		t.Run(test.header, func(t *testing.T) {
			assert.Equal(t, test.want, cleanHeaderText(test.header))
		})
	}
}

func TestSectionURL(t *testing.T) {
	assert.Equal(t,
		"https://docs.omnistrate.com/spec-guides/plan-spec/#compute-schema",
		sectionURL("https://docs.omnistrate.com/spec-guides/plan-spec/index.md", "`Compute schema`"))
	assert.Equal(t,
		"https://docs.omnistrate.com/build-guides/compose-spec/#x-omnistrate-capabilitiessidecars",
		sectionURL("https://docs.omnistrate.com/build-guides/compose-spec/index.md", "x-omnistrate-capabilities.sidecars"))
}

func TestMatchesTag(t *testing.T) {
	tests := []struct {
		name   string
		header string
		tag    string
		want   bool
	}{
		{"exact", "x-omnistrate-compute", "x-omnistrate-compute", true},
		{"case insensitive", "x-omnistrate-compute", "X-Omnistrate-Compute", true},
		{"substring", "x-omnistrate-capabilities.sidecars", "sidecars", true},
		{"prefix matches nested tag", "x-omnistrate-compute.instanceTypes", "x-omnistrate-compute", true},
		{"code span in header", "`Compute schema`", "Compute schema", true},
		{"code span in tag", "Compute schema", "`Compute schema`", true},
		{"anchor slug as tag", "x-omnistrate-capabilities.sidecars", "x-omnistrate-capabilitiessidecars", true},
		// The heading is singular but specs and the schema API use the plural.
		{"spec spelling finds singular heading", "x-omnistrate-image-registry-attribute", "x-omnistrate-image-registry-attributes", true},
		{"heading spelling still matches", "x-omnistrate-image-registry-attribute", "x-omnistrate-image-registry-attribute", true},
		{"unrelated", "x-omnistrate-storage", "networks", false},
		{"alias does not widen unrelated matches", "x-omnistrate-storage", "x-omnistrate-image-registry-attributes", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, matchesTag(test.header, test.tag))
		})
	}
}

func TestFetchSpecSectionsFallsBackPastStubPage(t *testing.T) {
	const stub = "https://docs.example.com/old/compose-spec/index.md"
	const current = "https://docs.example.com/new/compose-spec/index.md"

	originalFetch := fetchDocumentationContent
	t.Cleanup(func() { fetchDocumentationContent = originalFetch })

	fetchDocumentationContent = func(url string) (string, error) {
		switch url {
		case stub:
			// A page left behind after the reference moved: prose, no tag headings.
			return "# Docker Compose Service Specification\n\n## What the Specification Covers\n\nSee the detailed reference.", nil
		case current:
			return "# Docker Compose Service Specification\n\n### x-omnistrate-compute\n\nCompute configuration.", nil
		default:
			return "", fmt.Errorf("unexpected url: %s", url)
		}
	}

	url, sections, err := fetchSpecSections(composeSpecName, []string{stub, current})
	require.NoError(t, err)
	assert.Equal(t, current, url)
	require.Len(t, sections, 1)
	assert.Equal(t, "x-omnistrate-compute", sections[0].Header)
}

func TestFetchSpecSectionsReportsEveryCandidate(t *testing.T) {
	originalFetch := fetchDocumentationContent
	t.Cleanup(func() { fetchDocumentationContent = originalFetch })

	fetchDocumentationContent = func(url string) (string, error) {
		if strings.HasSuffix(url, "unreachable/index.md") {
			return "", fmt.Errorf("HTTP 404")
		}
		return "# Title\n\nNo tag headings here.", nil
	}

	_, _, err := fetchSpecSections(composeSpecName, []string{
		"https://docs.example.com/unreachable/index.md",
		"https://docs.example.com/stub/index.md",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tag information found in the compose spec documentation")
	// Both candidates are named so a docs move is diagnosable from the error alone.
	assert.Contains(t, err.Error(), "unreachable/index.md")
	assert.Contains(t, err.Error(), "stub/index.md")
}

func TestSpecSectionListingStripsMarkupFromTags(t *testing.T) {
	originalFetch := fetchDocumentationContent
	t.Cleanup(func() { fetchDocumentationContent = originalFetch })

	fetchDocumentationContent = func(string) (string, error) {
		return "## Plan schema definitions\n\n### `Compute schema`\n\nCompute configuration.\n\n### Network schema\n\nNetwork configuration.", nil
	}

	tags, err := ListPlanSpecSections()
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "Compute schema", tags[0].AvailableTag)
	assert.Equal(t, "Network schema", tags[1].AvailableTag)

	results, err := SearchPlanSpecSections("compute schema")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Compute schema", results[0].Tag)
	assert.Contains(t, results[0].URL, "#compute-schema")
}
