package instance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// formatTerraformPlan takes raw terraform plan JSON and returns a human-readable
// diff-like representation similar to `terraform plan` output.
// If the JSON cannot be parsed, the raw text is returned as-is.
func formatTerraformPlan(rawJSON string) string {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" {
		return ""
	}

	var plan tfPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		// Not valid JSON — return as-is
		return rawJSON
	}

	// If no resource_changes and no meaningful planned_values, return raw JSON (might not be a plan)
	hasOutputs := plan.PlannedValues != nil && len(plan.PlannedValues.Outputs) > 0
	hasResources := plan.PlannedValues != nil && plan.PlannedValues.RootModule != nil && len(plan.PlannedValues.RootModule.Resources) > 0
	if len(plan.ResourceChanges) == 0 && !hasOutputs && !hasResources {
		return rawJSON
	}

	var b strings.Builder

	// Header
	if plan.TerraformVersion != "" {
		b.WriteString(fmt.Sprintf("Terraform v%s\n", plan.TerraformVersion))
	}

	// Count changes by action
	var toCreate, toUpdate, toDelete, toReplace, noOp int
	for _, rc := range plan.ResourceChanges {
		switch classifyActions(rc.Change.Actions) {
		case actionCreate:
			toCreate++
		case actionUpdate:
			toUpdate++
		case actionDelete:
			toDelete++
		case actionReplace:
			toReplace++
		default:
			noOp++
		}
	}

	if toCreate+toUpdate+toDelete+toReplace == 0 && len(plan.ResourceChanges) > 0 {
		b.WriteString("\nNo changes. Infrastructure is up-to-date.\n")
		return b.String()
	}

	// Resource changes
	if len(plan.ResourceChanges) > 0 {
		b.WriteString("\n")
		for i, rc := range plan.ResourceChanges {
			if i > 0 {
				b.WriteString("\n")
			}
			writeResourceChange(&b, &rc)
		}
	}

	// Output changes
	if plan.PlannedValues != nil && len(plan.PlannedValues.Outputs) > 0 {
		b.WriteString("\n─── Outputs ───\n\n")
		outputKeys := make([]string, 0, len(plan.PlannedValues.Outputs))
		for k := range plan.PlannedValues.Outputs {
			outputKeys = append(outputKeys, k)
		}
		sort.Strings(outputKeys)
		for _, k := range outputKeys {
			out := plan.PlannedValues.Outputs[k]
			sensitiveTag := ""
			if out.Sensitive {
				sensitiveTag = " (sensitive)"
			}
			b.WriteString(fmt.Sprintf("  + %s%s\n", k, sensitiveTag))
		}
	}

	// Summary line
	b.WriteString(fmt.Sprintf("\nPlan: %d to add, %d to change, %d to destroy.\n",
		toCreate+toReplace, toUpdate, toDelete+toReplace))

	return b.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// Terraform Plan JSON structures (subset needed for formatting)
// ──────────────────────────────────────────────────────────────────────────────

type tfPlan struct {
	FormatVersion    string              `json:"format_version"`
	TerraformVersion string              `json:"terraform_version"`
	PlannedValues    *tfPlannedValues    `json:"planned_values"`
	ResourceChanges  []tfResourceChange  `json:"resource_changes"`
	OutputChanges    map[string]tfChange `json:"output_changes"`
}

type tfPlannedValues struct {
	Outputs    map[string]tfOutput `json:"outputs"`
	RootModule *tfModule           `json:"root_module"`
}

type tfOutput struct {
	Sensitive bool        `json:"sensitive"`
	Value     interface{} `json:"value,omitempty"`
}

type tfModule struct {
	Resources []tfResource `json:"resources"`
}

type tfResource struct {
	Address      string                 `json:"address"`
	Mode         string                 `json:"mode"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	ProviderName string                 `json:"provider_name"`
	Values       map[string]interface{} `json:"values"`
}

type tfResourceChange struct {
	Address      string   `json:"address"`
	Mode         string   `json:"mode"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	ProviderName string   `json:"provider_name"`
	Change       tfChange `json:"change"`
}

type tfChange struct {
	Actions         []string               `json:"actions"`
	Before          map[string]interface{} `json:"before"`
	After           map[string]interface{} `json:"after"`
	AfterUnknown    interface{}            `json:"after_unknown"`
	BeforeSensitive interface{}            `json:"before_sensitive"`
	AfterSensitive  interface{}            `json:"after_sensitive"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Action classification
// ──────────────────────────────────────────────────────────────────────────────

type changeAction int

const (
	actionNoOp changeAction = iota
	actionCreate
	actionUpdate
	actionDelete
	actionReplace
)

func classifyActions(actions []string) changeAction {
	if len(actions) == 1 {
		switch actions[0] {
		case "create":
			return actionCreate
		case "update":
			return actionUpdate
		case "delete":
			return actionDelete
		case "no-op", "read":
			return actionNoOp
		}
	}
	if len(actions) == 2 {
		// ["delete", "create"] or ["create", "delete"] = replace
		hasCreate := false
		hasDelete := false
		for _, a := range actions {
			if a == "create" {
				hasCreate = true
			}
			if a == "delete" {
				hasDelete = true
			}
		}
		if hasCreate && hasDelete {
			return actionReplace
		}
	}
	return actionNoOp
}

// ──────────────────────────────────────────────────────────────────────────────
// Resource change formatting
// ──────────────────────────────────────────────────────────────────────────────

func writeResourceChange(b *strings.Builder, rc *tfResourceChange) {
	action := classifyActions(rc.Change.Actions)

	var prefix, verb string
	switch action {
	case actionCreate:
		prefix = "+"
		verb = "will be created"
	case actionUpdate:
		prefix = "~"
		verb = "will be updated in-place"
	case actionDelete:
		prefix = "-"
		verb = "will be destroyed"
	case actionReplace:
		prefix = "-/+"
		verb = "must be replaced"
	default:
		return // skip no-op
	}

	// Header: # aws_db_instance.example1 will be created
	b.WriteString(fmt.Sprintf("  # %s %s\n", rc.Address, verb))
	b.WriteString(fmt.Sprintf("  %s resource %q %q {\n", prefix, rc.Type, rc.Name))

	// Collect after-sensitive map for masking
	afterSensitive := toStringBoolMap(rc.Change.AfterSensitive)
	afterUnknown := toStringBoolMap(rc.Change.AfterUnknown)

	switch action {
	case actionCreate:
		writeCreateAttributes(b, rc.Change.After, afterSensitive, afterUnknown, prefix)
	case actionDelete:
		writeDeleteAttributes(b, rc.Change.Before, prefix)
	case actionUpdate:
		writeUpdateAttributes(b, rc.Change.Before, rc.Change.After, afterSensitive, afterUnknown)
	case actionReplace:
		writeUpdateAttributes(b, rc.Change.Before, rc.Change.After, afterSensitive, afterUnknown)
	}

	b.WriteString("    }\n")
}

func writeCreateAttributes(b *strings.Builder, after map[string]interface{}, sensitive, unknown map[string]bool, prefix string) {
	// Merge keys from after + unknown (unknown may have keys not in after)
	allKeys := make(map[string]bool)
	for k := range after {
		allKeys[k] = true
	}
	for k := range unknown {
		allKeys[k] = true
	}
	if len(allKeys) == 0 {
		return
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	maxKeyLen := maxKeyLength(keys)

	for _, k := range keys {
		v, inAfter := after[k]

		// Skip null values and complex nested types for cleaner output
		if inAfter && v == nil {
			continue
		}
		if inAfter && isComplexEmpty(v) {
			continue
		}

		padding := strings.Repeat(" ", maxKeyLen-len(k))
		if sensitive[k] {
			b.WriteString(fmt.Sprintf("      %s %-s%s = (sensitive value)\n", prefix, k, padding))
		} else if unknown[k] {
			b.WriteString(fmt.Sprintf("      %s %-s%s = (known after apply)\n", prefix, k, padding))
		} else if inAfter {
			b.WriteString(fmt.Sprintf("      %s %-s%s = %s\n", prefix, k, padding, formatValue(v)))
		}
	}
}

func writeDeleteAttributes(b *strings.Builder, before map[string]interface{}, prefix string) {
	if len(before) == 0 {
		return
	}

	keys := sortedKeys(before)
	maxKeyLen := maxKeyLength(keys)

	for _, k := range keys {
		v := before[k]
		if v == nil {
			continue
		}
		if isComplexEmpty(v) {
			continue
		}
		padding := strings.Repeat(" ", maxKeyLen-len(k))
		b.WriteString(fmt.Sprintf("      %s %-s%s = %s\n", prefix, k, padding, formatValue(v)))
	}
}

func writeUpdateAttributes(b *strings.Builder, before, after map[string]interface{}, sensitive, unknown map[string]bool) {
	// Merge all keys
	allKeys := make(map[string]bool)
	for k := range before {
		allKeys[k] = true
	}
	for k := range after {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	maxKeyLen := maxKeyLength(keys)

	for _, k := range keys {
		beforeVal, hadBefore := before[k]
		afterVal, hasAfter := after[k]
		padding := strings.Repeat(" ", maxKeyLen-len(k))

		if !hadBefore && hasAfter {
			// Added
			if sensitive[k] {
				b.WriteString(fmt.Sprintf("      + %-s%s = (sensitive value)\n", k, padding))
			} else if unknown[k] {
				b.WriteString(fmt.Sprintf("      + %-s%s = (known after apply)\n", k, padding))
			} else {
				b.WriteString(fmt.Sprintf("      + %-s%s = %s\n", k, padding, formatValue(afterVal)))
			}
		} else if hadBefore && !hasAfter {
			// Removed
			b.WriteString(fmt.Sprintf("      - %-s%s = %s\n", k, padding, formatValue(beforeVal)))
		} else {
			// Check if changed
			beforeStr := formatValue(beforeVal)
			afterStr := formatValue(afterVal)
			if sensitive[k] {
				afterStr = "(sensitive value)"
			} else if unknown[k] {
				afterStr = "(known after apply)"
			}
			if beforeStr != afterStr {
				b.WriteString(fmt.Sprintf("      ~ %-s%s = %s -> %s\n", k, padding, beforeStr, afterStr))
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Value formatting helpers
// ──────────────────────────────────────────────────────────────────────────────

func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item)
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	case map[string]interface{}:
		if len(val) == 0 {
			return "{}"
		}
		keys := sortedKeys(val)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = fmt.Sprintf("%s = %s", k, formatValue(val[k]))
		}
		return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toStringBoolMap converts an interface{} to a map[string]bool.
// Terraform plan JSON uses this for after_sensitive and after_unknown.
// The value can be a bool (applies to whole resource) or a map of field→bool.
func toStringBoolMap(v interface{}) map[string]bool {
	result := make(map[string]bool)
	if v == nil {
		return result
	}
	switch val := v.(type) {
	case map[string]interface{}:
		for k, sv := range val {
			if b, ok := sv.(bool); ok && b {
				result[k] = true
			}
		}
	}
	return result
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func maxKeyLength(keys []string) int {
	max := 0
	for _, k := range keys {
		if len(k) > max {
			max = len(k)
		}
	}
	return max
}

// isComplexEmpty returns true if the value is an empty slice or empty map.
func isComplexEmpty(v interface{}) bool {
	switch val := v.(type) {
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return false
}
