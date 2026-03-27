package instance

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatTerraformPlan_EmptyString(t *testing.T) {
	result := formatTerraformPlan("")
	require.Equal(t, "", result)
}

func TestFormatTerraformPlan_NonJSON(t *testing.T) {
	raw := "Error: Failed to refresh state"
	result := formatTerraformPlan(raw)
	require.Equal(t, raw, result)
}

func TestFormatTerraformPlan_InvalidJSON(t *testing.T) {
	raw := `{"broken json`
	result := formatTerraformPlan(raw)
	require.Equal(t, raw, result)
}

func TestFormatTerraformPlan_NoResourceChanges(t *testing.T) {
	raw := `{"format_version":"1.2","planned_values":{}}`
	result := formatTerraformPlan(raw)
	// No resource_changes, no outputs, no resources — should return raw JSON
	require.Equal(t, raw, result)
}

func TestFormatTerraformPlan_EmptyResourceChangesOnly(t *testing.T) {
	raw := `{"format_version":"1.2","planned_values":{"outputs":{"db":{"sensitive":false}}}}`
	result := formatTerraformPlan(raw)
	// Has outputs but no resource_changes — should still format
	require.Contains(t, result, "─── Outputs ───")
	require.Contains(t, result, "+ db")
}

func TestFormatTerraformPlan_CreateResources(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"planned_values": {
			"outputs": {
				"db_endpoint": {"sensitive": false},
				"db_password": {"sensitive": true}
			},
			"root_module": {
				"resources": [{
					"address": "aws_db_instance.main",
					"type": "aws_db_instance",
					"name": "main",
					"values": {"engine": "mysql"}
				}]
			}
		},
		"resource_changes": [{
			"address": "aws_db_instance.main",
			"type": "aws_db_instance",
			"name": "main",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {
					"allocated_storage": 20,
					"engine": "mysql",
					"engine_version": "8.0.44",
					"identifier": "my-db",
					"instance_class": "db.t3.micro",
					"password": "secret",
					"username": "admin",
					"skip_final_snapshot": true
				},
				"after_unknown": {
					"arn": true,
					"endpoint": true,
					"id": true
				},
				"after_sensitive": {
					"password": true
				}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "Terraform v1.11.5")
	require.Contains(t, result, "# aws_db_instance.main will be created")
	require.Contains(t, result, `+ resource "aws_db_instance" "main"`)
	require.Contains(t, result, `+ engine`)
	require.Contains(t, result, `"mysql"`)
	require.Contains(t, result, `+ password`)
	require.Contains(t, result, "(sensitive value)")
	require.Contains(t, result, `+ arn`)
	require.Contains(t, result, "(known after apply)")
	require.Contains(t, result, "Plan: 1 to add, 0 to change, 0 to destroy.")

	// Outputs section
	require.Contains(t, result, "─── Outputs ───")
	require.Contains(t, result, "+ db_endpoint")
	require.Contains(t, result, "+ db_password (sensitive)")
}

func TestFormatTerraformPlan_DeleteResource(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["delete"],
				"before": {
					"ami": "ami-12345678",
					"instance_type": "t2.micro"
				},
				"after": null
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "# aws_instance.web will be destroyed")
	require.Contains(t, result, `- resource "aws_instance" "web"`)
	require.Contains(t, result, "Plan: 0 to add, 0 to change, 1 to destroy.")
}

func TestFormatTerraformPlan_UpdateResource(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["update"],
				"before": {
					"instance_type": "t2.micro",
					"ami": "ami-12345678"
				},
				"after": {
					"instance_type": "t2.small",
					"ami": "ami-12345678"
				}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "# aws_instance.web will be updated in-place")
	require.Contains(t, result, `~ resource "aws_instance" "web"`)
	require.Contains(t, result, `~ instance_type`)
	require.Contains(t, result, `"t2.micro"`)
	require.Contains(t, result, `"t2.small"`)
	require.Contains(t, result, "Plan: 0 to add, 1 to change, 0 to destroy.")
}

func TestFormatTerraformPlan_ReplaceResource(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["delete", "create"],
				"before": {"ami": "old"},
				"after": {"ami": "new"}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "# aws_instance.web must be replaced")
	require.Contains(t, result, `-/+ resource "aws_instance" "web"`)
	require.Contains(t, result, "Plan: 1 to add, 0 to change, 1 to destroy.")
}

func TestFormatTerraformPlan_NoOpIsUpToDate(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["no-op"],
				"before": {"ami": "same"},
				"after": {"ami": "same"}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "No changes. Infrastructure is up-to-date.")
}

func TestFormatTerraformPlan_NullValuesSkipped(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {
					"ami": "ami-123",
					"domain": null,
					"tags": null
				}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, `+ ami`)
	// null values should be skipped
	require.NotContains(t, result, "domain")
	require.NotContains(t, result, "tags")
}

func TestFormatTerraformPlan_MultipleResources(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [
			{
				"address": "aws_security_group.sg",
				"type": "aws_security_group",
				"name": "sg",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"name": "my-sg", "vpc_id": "vpc-123"}
				}
			},
			{
				"address": "aws_db_instance.main",
				"type": "aws_db_instance",
				"name": "main",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"engine": "mysql"}
				}
			}
		]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "# aws_security_group.sg will be created")
	require.Contains(t, result, "# aws_db_instance.main will be created")
	require.Contains(t, result, "Plan: 2 to add, 0 to change, 0 to destroy.")
}

func TestClassifyActions(t *testing.T) {
	require.Equal(t, actionCreate, classifyActions([]string{"create"}))
	require.Equal(t, actionUpdate, classifyActions([]string{"update"}))
	require.Equal(t, actionDelete, classifyActions([]string{"delete"}))
	require.Equal(t, actionNoOp, classifyActions([]string{"no-op"}))
	require.Equal(t, actionNoOp, classifyActions([]string{"read"}))
	require.Equal(t, actionReplace, classifyActions([]string{"delete", "create"}))
	require.Equal(t, actionReplace, classifyActions([]string{"create", "delete"}))
}

func TestFormatValue(t *testing.T) {
	require.Equal(t, `"hello"`, formatValue("hello"))
	require.Equal(t, "42", formatValue(float64(42)))
	require.Equal(t, "3.14", formatValue(float64(3.14)))
	require.Equal(t, "true", formatValue(true))
	require.Equal(t, "false", formatValue(false))
	require.Equal(t, "null", formatValue(nil))
	require.Equal(t, "[]", formatValue([]interface{}{}))
	require.Equal(t, `["a", "b"]`, formatValue([]interface{}{"a", "b"}))
	require.Equal(t, "{}", formatValue(map[string]interface{}{}))
}

func TestToStringBoolMap(t *testing.T) {
	// nil input
	result, allFlagged := toStringBoolMap(nil)
	require.Empty(t, result)
	require.False(t, allFlagged)

	// bool(true) input — whole-resource flag
	result, allFlagged = toStringBoolMap(true)
	require.Empty(t, result)
	require.True(t, allFlagged)

	// bool(false) input
	result, allFlagged = toStringBoolMap(false)
	require.Empty(t, result)
	require.False(t, allFlagged)

	// map input
	result, allFlagged = toStringBoolMap(map[string]interface{}{
		"password": true,
		"username": false,
		"tags":     map[string]interface{}{}, // not a bool
	})
	require.False(t, allFlagged)
	require.True(t, result["password"])
	require.False(t, result["username"])
	require.False(t, result["tags"])
}

func TestFormatTerraformPlan_RealWorldExample(t *testing.T) {
	// Simplified version of the real-world plan from the problem statement
	raw := `{"format_version":"1.2","terraform_version":"1.11.5","variables":{"vpc_id":{"value":"vpc-0bab56277def0b464"}},"planned_values":{"outputs":{"db_endpoints_1":{"sensitive":false},"db_endpoints_2":{"sensitive":true}},"root_module":{"resources":[{"address":"aws_db_instance.example1","mode":"managed","type":"aws_db_instance","name":"example1","values":{"allocated_storage":20,"engine":"mysql","identifier":"e2e-instance-1","instance_class":"db.t3.micro","password":"yourpassword","username":"admin"}}]}},"resource_changes":[{"address":"aws_db_instance.example1","mode":"managed","type":"aws_db_instance","name":"example1","change":{"actions":["create"],"before":null,"after":{"allocated_storage":20,"engine":"mysql","engine_version":"8.0.44","identifier":"e2e-instance-1","instance_class":"db.t3.micro","password":"yourpassword","username":"admin"},"after_unknown":{"arn":true,"endpoint":true,"id":true},"after_sensitive":{"password":true}}},{"address":"aws_security_group.rds_sg","mode":"managed","type":"aws_security_group","name":"rds_sg","change":{"actions":["create"],"before":null,"after":{"description":"Security group for RDS instances","name":"e2e-rds-security-group","vpc_id":"vpc-0bab56277def0b464"},"after_unknown":{"id":true}}}]}`

	result := formatTerraformPlan(raw)

	// Should be formatted, not raw JSON
	require.NotContains(t, result, `"format_version"`)

	// Should contain readable output
	require.Contains(t, result, "Terraform v1.11.5")
	require.Contains(t, result, "# aws_db_instance.example1 will be created")
	require.Contains(t, result, "# aws_security_group.rds_sg will be created")
	require.Contains(t, result, "(sensitive value)")
	require.Contains(t, result, "(known after apply)")
	require.Contains(t, result, "─── Outputs ───")
	require.Contains(t, result, "db_endpoints_1")
	require.Contains(t, result, "db_endpoints_2 (sensitive)")
	require.Contains(t, result, "Plan: 2 to add, 0 to change, 0 to destroy.")

	// Verify it's multi-line and readable
	lines := strings.Split(result, "\n")
	require.Greater(t, len(lines), 10, "formatted plan should have multiple lines")
}

func TestFormatTerraformPlan_AllSensitive(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_db_instance.secret",
			"type": "aws_db_instance",
			"name": "secret",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {
					"password": "super-secret",
					"username": "admin"
				},
				"after_unknown": {},
				"after_sensitive": true
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "(sensitive value)")
	require.NotContains(t, result, "super-secret")
	require.NotContains(t, result, `"admin"`)
	require.Contains(t, result, "Plan: 1 to add, 0 to change, 0 to destroy.")
}

func TestFormatTerraformPlan_AllUnknown(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_instance.web",
			"type": "aws_instance",
			"name": "web",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {
					"ami": "ami-123",
					"id": "placeholder"
				},
				"after_unknown": true,
				"after_sensitive": {}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "(known after apply)")
	require.NotContains(t, result, `"ami-123"`)
	require.NotContains(t, result, `"placeholder"`)
}

func TestFormatTerraformPlan_AllSensitiveUpdate(t *testing.T) {
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_db_instance.secret",
			"type": "aws_db_instance",
			"name": "secret",
			"change": {
				"actions": ["update"],
				"before": {
					"password": "old-secret",
					"username": "admin"
				},
				"after": {
					"password": "new-secret",
					"username": "admin"
				},
				"after_unknown": {},
				"after_sensitive": true
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "(sensitive value)")
	require.NotContains(t, result, "new-secret")
	require.Contains(t, result, "Plan: 0 to add, 1 to change, 0 to destroy.")
}
