package dataaccess

import (
	"fmt"
	"sort"
	"strings"
)

// Spec names used in user-facing messages and to pick the schema fallback that
// applies to a given documentation page.
const (
	composeSpecName = "compose spec"
	planSpecName    = "plan spec"
)

// JSONSchemaTypeInfo describes one schema type accepted by the JSON schema API.
type JSONSchemaTypeInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// jsonSchemaTypes are the types the JSON schema API serves. Every entry here is
// verified to return a schema; the API's allow list is wider, but
// deployment-cell-amenities and the orchestration DSL types currently answer
// "schema not found", so listing them would only send callers down a dead end.
// Unlisted types are still passed through to the API rather than rejected locally.
var jsonSchemaTypes = []JSONSchemaTypeInfo{
	{Type: "compose", Description: "Full Docker Compose spec schema, including every Omnistrate extension"},
	{Type: "service-plan", Description: "Full Plan spec (ServicePlanSpec) schema for Helm, operator, Kustomize and Terraform plans"},
	{Type: "system-parameters", Description: "System parameters available for variable interpolation"},
	{Type: "x-omnistrate-service-plan", Description: "Plan, tenancy, deployment model and feature configuration"},
	{Type: "x-omnistrate-api-params", Description: "Customer-facing API parameters"},
	{Type: "x-omnistrate-compute", Description: "Instance types, root volume and CPU architecture"},
	{Type: "x-omnistrate-storage", Description: "Storage volume configuration"},
	{Type: "x-omnistrate-capabilities", Description: "Autoscaling, backups, custom DNS, sidecars and other capabilities"},
	{Type: "x-omnistrate-actionhooks", Description: "Lifecycle action hooks"},
	{Type: "x-omnistrate-job-config", Description: "Job resource configuration"},
	{Type: "x-omnistrate-load-balancer", Description: "L4 and L7 load balancer configuration"},
	{Type: "x-omnistrate-image-registry-attributes", Description: "Private image registry credentials and attributes"},
	{Type: "x-omnistrate-mode-internal", Description: "Marks a resource as internal rather than customer-facing"},
	{Type: "x-omnistrate-proxy-type", Description: "Proxy type for a resource"},
	{Type: "x-omnistrate-integrations", Description: "Logging, metrics and licensing integrations (deprecated)"},
	{Type: "x-customer-integrations", Description: "Customer-scoped logging, metrics and licensing integrations"},
	{Type: "x-internal-integrations", Description: "Internal logging and metrics integrations"},
}

// docHeadingAliases maps the spelling used in a real spec, and by the schema API,
// onto the spelling the documentation heading uses, for the cases where the two have
// drifted apart. Without this, searching for the name you actually write in a spec
// matches no heading and the caller gets the full tag listing instead of the section.
var docHeadingAliases = map[string]string{
	"x-omnistrate-image-registry-attributes": "x-omnistrate-image-registry-attribute",
}

// jsonSchemaTypeAliases maps documentation heading spellings onto the schema type
// the API actually accepts. Doc headings and API type names drift apart
// occasionally; this keeps a tag copied straight out of the docs usable.
var jsonSchemaTypeAliases = map[string]string{
	"x-omnistrate-image-registry-attribute": "x-omnistrate-image-registry-attributes",
	"x-omnistrate-action-hooks":             "x-omnistrate-actionhooks",
	"x-omnistrate-api-parameters":           "x-omnistrate-api-params",
	"compose-spec":                          "compose",
	"plan-spec":                             "service-plan",
	"service-plan-spec":                     "service-plan",
	"serviceplanspec":                       "service-plan",
}

// ListJSONSchemaTypes returns the schema types the JSON schema API accepts,
// sorted by type name.
func ListJSONSchemaTypes() []JSONSchemaTypeInfo {
	types := make([]JSONSchemaTypeInfo, len(jsonSchemaTypes))
	copy(types, jsonSchemaTypes)
	sort.Slice(types, func(i, j int) bool { return types[i].Type < types[j].Type })
	return types
}

// IsValidJSONSchemaType reports whether the API accepts the given schema type.
func IsValidJSONSchemaType(schemaType string) bool {
	for _, known := range jsonSchemaTypes {
		if known.Type == schemaType {
			return true
		}
	}
	return false
}

// jsonSchemaTypeNames returns just the type names, sorted, for error messages.
func jsonSchemaTypeNames() []string {
	types := ListJSONSchemaTypes()
	names := make([]string, 0, len(types))
	for _, t := range types {
		names = append(names, t.Type)
	}
	return names
}

// ErrUnknownJSONSchemaType describes a tag that could not be mapped onto a schema
// type the API serves, and lists the types that are available.
type ErrUnknownJSONSchemaType struct {
	Tag string
}

func (e *ErrUnknownJSONSchemaType) Error() string {
	return fmt.Sprintf("no JSON schema is published for %q; pass one of the following types to 'omnistrate-ctl docs json-schema' instead: %s",
		e.Tag, strings.Join(jsonSchemaTypeNames(), ", "))
}

// ResolveComposeSpecJSONSchemaType maps a compose spec documentation tag onto the
// JSON schema type to request from the API.
func ResolveComposeSpecJSONSchemaType(tag string) (string, error) {
	return resolveJSONSchemaType(composeSpecName, tag)
}

// ResolvePlanSpecJSONSchemaType maps a plan spec documentation tag onto the JSON
// schema type to request from the API.
func ResolvePlanSpecJSONSchemaType(tag string) (string, error) {
	return resolveJSONSchemaType(planSpecName, tag)
}

// resolveJSONSchemaType maps a documentation tag onto the schema type to request
// from the API. Documentation headings are not schema type names: they carry code
// spans, address nested fields with dotted paths, and on the plan spec page they
// name sub-schemas of a single top-level schema. specName selects the fallback
// applied when a tag is not itself an extension name.
//
// Resolution order:
//  1. the cleaned, lowercased tag if the API serves it directly
//  2. a known alias for that tag
//  3. for an "x-" extension tag, the root extension of a dotted path
//     (x-omnistrate-capabilities.sidecars resolves to x-omnistrate-capabilities)
//  4. for plan spec tags, the full service-plan schema, since every plan spec
//     heading documents a definition inside it
//
// Anything else returns *ErrUnknownJSONSchemaType rather than guessing.
func resolveJSONSchemaType(specName, tag string) (string, error) {
	normalized := strings.ToLower(cleanHeaderText(tag))
	// Headings such as "x-omnistrate-my-account (deprecated)" carry a trailing note.
	if idx := strings.Index(normalized, "("); idx > 0 {
		normalized = strings.TrimSpace(normalized[:idx])
	}

	if normalized == "" {
		return "", &ErrUnknownJSONSchemaType{Tag: tag}
	}

	if IsValidJSONSchemaType(normalized) {
		return normalized, nil
	}

	if alias, ok := jsonSchemaTypeAliases[normalized]; ok {
		return alias, nil
	}

	// Nested extension fields are documented as dotted paths under their root
	// extension, which is where the schema lives.
	if strings.HasPrefix(normalized, "x-") {
		root := normalized
		if idx := strings.Index(root, "."); idx > 0 {
			root = root[:idx]
		}
		if IsValidJSONSchemaType(root) {
			return root, nil
		}
		if alias, ok := jsonSchemaTypeAliases[root]; ok {
			return alias, nil
		}
		return "", &ErrUnknownJSONSchemaType{Tag: tag}
	}

	// Every plan spec heading documents a definition inside the service-plan
	// schema, so that is the schema that covers the tag.
	if specName == planSpecName {
		return "service-plan", nil
	}

	return "", &ErrUnknownJSONSchemaType{Tag: tag}
}
