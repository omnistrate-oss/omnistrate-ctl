package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

// SpecViolation is a single schema violation found in a spec file.
type SpecViolation struct {
	// Path is the location in the spec, as a slash-delimited pointer such as
	// "/services/0/apiParameters/2/type". "/" means the root of the document.
	Path string `json:"path"`
	// Message is the validator's explanation of why that location is invalid.
	Message string `json:"message"`
}

// SpecValidationResult reports whether a spec satisfies its schema.
type SpecValidationResult struct {
	File       string          `json:"file"`
	SpecType   string          `json:"spec_type"`
	Valid      bool            `json:"valid"`
	Violations []SpecViolation `json:"violations,omitempty"`
}

// DetectSpecType guesses which schema a spec file should be validated against.
//
// The two shapes are distinguishable at the root: a Compose file keys its services
// by name (a mapping) and carries `x-omnistrate-*` extensions, while a
// ServicePlanSpec lists services (a sequence) and puts plan fields such as
// `helmChartConfiguration` inside them. Returns "" when neither shape is evident,
// so the caller can ask for --spec-type rather than guessing wrong.
func DetectSpecType(specYAML []byte) string {
	var root map[string]json.RawMessage
	jsonBytes, err := yaml.YAMLToJSON(specYAML)
	if err != nil {
		return ""
	}
	if err := json.Unmarshal(jsonBytes, &root); err != nil {
		return ""
	}

	for key := range root {
		if strings.HasPrefix(key, "x-omnistrate-") || strings.HasPrefix(key, "x-customer-") ||
			strings.HasPrefix(key, "x-internal-") {
			return "compose"
		}
	}

	if rawServices, ok := root["services"]; ok {
		trimmed := bytes.TrimLeft(rawServices, " \t\r\n")
		if len(trimmed) > 0 {
			switch trimmed[0] {
			case '[':
				// A sequence of services is the ServicePlanSpec shape.
				return "service-plan"
			case '{':
				// Services keyed by name is the Compose shape.
				return "compose"
			}
		}
	}

	// A bare `name:` plus any of these plan-only keys is still a plan spec.
	for _, key := range []string{"systemWorkflows", "customWorkflows", "sharedFileSystems", "loadBalancers"} {
		if _, ok := root[key]; ok {
			return "service-plan"
		}
	}

	return ""
}

// ValidateSpec checks a compose or ServicePlanSpec YAML document against the
// authoritative JSON schema served by the platform. specType must be a type the
// schema API accepts (see ListJSONSchemaTypes).
func ValidateSpec(ctx context.Context, fileName string, specYAML []byte, specType string) (*SpecValidationResult, error) {
	rawSchema, err := GetJSONSchema(ctx, specType)
	if err != nil {
		return nil, err
	}

	return validateSpecAgainstSchema(fileName, specYAML, specType, rawSchema)
}

// ValidateSpecWithSchemaFile validates against a schema already on disk instead of
// fetching one. Use it to run fully offline, to pin a schema in CI, or to check a spec
// against a schema version that is not deployed yet.
func ValidateSpecWithSchemaFile(fileName string, specYAML []byte, specType, schemaPath string) (*SpecValidationResult, error) {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var rawSchema interface{}
	if err = json.Unmarshal(schemaBytes, &rawSchema); err != nil {
		return nil, fmt.Errorf("failed to parse the schema at %s: %w", schemaPath, err)
	}

	if specType == "" {
		specType = schemaPath
	}

	return validateSpecAgainstSchema(fileName, specYAML, specType, rawSchema)
}

func validateSpecAgainstSchema(fileName string, specYAML []byte, specType string, rawSchema interface{}) (*SpecValidationResult, error) {
	compiled, err := compileSchema(specType, rawSchema)
	if err != nil {
		return nil, err
	}

	// Mirror how the platform reads specs: YAML is converted to JSON and then
	// unmarshalled with encoding/json semantics.
	jsonBytes, err := yaml.YAMLToJSON(specYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s as YAML: %w", fileName, err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fileName, err)
	}

	result := &SpecValidationResult{File: fileName, SpecType: specType}

	if err = compiled.Validate(instance); err != nil {
		validationErr := new(*jsonschema.ValidationError)
		if !errors.As(err, validationErr) {
			return nil, err
		}
		// DetailedOutput keeps the cause tree, so the walk below can reach the unit
		// that names the actual offending field. BasicOutput flattens it and loses
		// those causes behind a generic "validation failed".
		result.Violations = collectViolations((*validationErr).DetailedOutput())
		return result, nil
	}

	result.Valid = true
	return result, nil
}

// compileSchema turns the schema document returned by the API into a compiled
// validator. The schema declares its own draft via $schema, so the compiler picks
// the right dialect (the compose schema is 2019-09, the plan schema is 2020-12).
func compileSchema(specType string, rawSchema interface{}) (*jsonschema.Schema, error) {
	schemaBytes, err := json.Marshal(rawSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to read the %s schema: %w", specType, err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the %s schema: %w", specType, err)
	}

	// A stable in-memory URL so $ref pointers inside the document resolve without
	// the compiler trying to fetch anything over the network.
	resourceURL := fmt.Sprintf("https://omnistrate.local/%s-schema.json", specType)

	compiler := jsonschema.NewCompiler()
	if err = compiler.AddResource(resourceURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to load the %s schema: %w", specType, err)
	}

	compiled, err := compiler.Compile(resourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile the %s schema: %w", specType, err)
	}

	return compiled, nil
}

// collectViolations flattens the validator's output tree down to its leaves, which
// are the units that name an actual offending field. Interior nodes only report that
// some subschema failed, which is noise for someone fixing a spec.
func collectViolations(root *jsonschema.OutputUnit) []SpecViolation {
	seen := make(map[string]struct{})
	var violations []SpecViolation

	var walk func(unit *jsonschema.OutputUnit)
	walk = func(unit *jsonschema.OutputUnit) {
		if len(unit.Errors) > 0 {
			for i := range unit.Errors {
				walk(&unit.Errors[i])
			}
			return
		}
		if unit.Error == nil {
			return
		}
		path := unit.InstanceLocation
		if path == "" {
			path = "/"
		}
		message := unit.Error.String()
		key := path + "\x00" + message
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			violations = append(violations, SpecViolation{Path: path, Message: message})
		}
	}
	walk(root)

	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		return violations[i].Message < violations[j].Message
	})

	return violations
}
