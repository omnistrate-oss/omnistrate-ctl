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

func TestFormatTerraformPlan_OutputChangesWithScalarValues(t *testing.T) {
	// Production plan JSON contains output_changes where before/after are scalars (string, bool),
	// not maps. This previously caused json.Unmarshal to fail on the entire plan because
	// tfChange.Before/After were typed as map[string]interface{}.
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"planned_values": {
			"outputs": {
				"pubsub_id": {"sensitive": false},
				"redis_endpoint": {"sensitive": false}
			},
			"root_module": {
				"resources": [{
					"address": "aws_elasticache_cluster.example",
					"type": "aws_elasticache_cluster",
					"name": "example",
					"values": {"engine": "memcached"}
				}]
			}
		},
		"resource_changes": [{
			"address": "aws_elasticache_cluster.example",
			"type": "aws_elasticache_cluster",
			"name": "example",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {"engine": "memcached", "node_type": "cache.t3.micro"},
				"after_unknown": {"id": true, "arn": true},
				"before_sensitive": false,
				"after_sensitive": {"tags_all": {}}
			}
		}],
		"output_changes": {
			"pubsub_id": {
				"actions": ["create"],
				"before": null,
				"after": "",
				"after_unknown": false,
				"before_sensitive": false,
				"after_sensitive": false
			},
			"redis_endpoint": {
				"actions": ["create"],
				"before": null,
				"after_unknown": true,
				"before_sensitive": false,
				"after_sensitive": false
			}
		}
	}`

	result := formatTerraformPlan(raw)

	// Must NOT return raw JSON — the scalar output_changes must not break parsing
	require.NotContains(t, result, `"format_version"`, "formatter returned raw JSON — output_changes with scalar values broke unmarshal")

	require.Contains(t, result, "Terraform v1.11.5")
	require.Contains(t, result, "# aws_elasticache_cluster.example will be created")
	require.Contains(t, result, `+ resource "aws_elasticache_cluster" "example"`)
	require.Contains(t, result, "(known after apply)")
	require.Contains(t, result, "─── Outputs ───")
	require.Contains(t, result, "Plan: 1 to add, 0 to change, 0 to destroy.")
}

func TestFormatTerraformPlan_FullProductionPlan(t *testing.T) {
	// Realistic full production plan with output_changes, prior_state, configuration, etc.
	raw := `{"format_version":"1.2","terraform_version":"1.11.5","planned_values":{"outputs":{"pubsub_id":{"sensitive":false}}},"resource_changes":[{"address":"google_pubsub_topic.pubsub_app_topic","mode":"managed","type":"google_pubsub_topic","name":"pubsub_app_topic","provider_name":"registry.opentofu.org/hashicorp/google","change":{"actions":["create"],"before":null,"after":{"name":"test-topic","project":"dev"},"after_unknown":{"effective_labels":true,"id":true},"before_sensitive":false,"after_sensitive":{"effective_labels":{}}}}],"output_changes":{"pubsub_id":{"actions":["create"],"before":null,"after":"","after_unknown":false,"before_sensitive":false,"after_sensitive":false}},"prior_state":{"format_version":"1.0","terraform_version":"1.11.5","values":{"root_module":{}}},"configuration":{"root_module":{"resources":[{"address":"google_pubsub_topic.pubsub_app_topic","mode":"managed","type":"google_pubsub_topic","name":"pubsub_app_topic"}]}}}`

	result := formatTerraformPlan(raw)

	// Must be formatted, not raw JSON
	require.NotContains(t, result, `"format_version"`, "formatter returned raw JSON unchanged")

	require.Contains(t, result, "Terraform v1.11.5")
	require.Contains(t, result, "# google_pubsub_topic.pubsub_app_topic will be created")
	require.Contains(t, result, `"test-topic"`)
	require.Contains(t, result, "(known after apply)")
	require.Contains(t, result, "Plan: 1 to add, 0 to change, 0 to destroy.")
}

func TestToStringInterfaceMap(t *testing.T) {
	// nil returns nil
	require.Nil(t, toStringInterfaceMap(nil))

	// map returns map
	m := map[string]interface{}{"key": "value"}
	require.Equal(t, m, toStringInterfaceMap(m))

	// string returns nil (output_changes case)
	require.Nil(t, toStringInterfaceMap("some string"))

	// bool returns nil
	require.Nil(t, toStringInterfaceMap(true))

	// number returns nil
	require.Nil(t, toStringInterfaceMap(float64(42)))
}

func TestFormatTerraformPlan_ProductionComplexTypes(t *testing.T) {
	// Full production plan with:
	// - nested arrays in after (security group egress/ingress rules)
	// - nested arrays/maps in after_unknown (e.g., egress: [{cidr_blocks:[false]}])
	// - nested arrays/maps in after_sensitive (e.g., tags_all:{})
	// - before_sensitive as bool false
	// - output_changes with scalar before/after (string "", bool, null)
	// - multiple resource types in a single plan
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"planned_values": {
			"outputs": {
				"pubsub_id": {"sensitive": false, "type": "string", "value": ""},
				"redis_endpoint": {"sensitive": false}
			},
			"root_module": {
				"resources": [
					{
						"address": "aws_elasticache_cluster.example_memcached",
						"type": "aws_elasticache_cluster",
						"name": "example_memcached",
						"values": {"engine": "memcached", "num_cache_nodes": 2}
					},
					{
						"address": "aws_security_group.elasticache_sg",
						"type": "aws_security_group",
						"name": "elasticache_sg",
						"values": {
							"name": "e2e-sg",
							"egress": [{"cidr_blocks": ["0.0.0.0/0"], "from_port": 0, "protocol": "-1", "to_port": 0}],
							"ingress": [
								{"cidr_blocks": ["0.0.0.0/0"], "from_port": 11211, "protocol": "tcp", "to_port": 11211},
								{"cidr_blocks": ["0.0.0.0/0"], "from_port": 3306, "protocol": "tcp", "to_port": 3306}
							]
						}
					}
				]
			}
		},
		"resource_changes": [
			{
				"address": "aws_elasticache_cluster.example_memcached",
				"type": "aws_elasticache_cluster",
				"name": "example_memcached",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {
						"cluster_id": "e2e-memcached",
						"engine": "memcached",
						"node_type": "cache.t3.micro",
						"num_cache_nodes": 2,
						"subnet_group_name": "e2e-subnet-group",
						"log_delivery_configuration": [],
						"tags": null
					},
					"after_unknown": {
						"arn": true,
						"id": true,
						"cache_nodes": true,
						"engine_version": true,
						"log_delivery_configuration": [],
						"security_group_ids": true,
						"tags_all": true
					},
					"before_sensitive": false,
					"after_sensitive": {
						"cache_nodes": [],
						"log_delivery_configuration": [],
						"security_group_ids": [],
						"tags_all": {}
					}
				}
			},
			{
				"address": "aws_security_group.elasticache_sg",
				"type": "aws_security_group",
				"name": "elasticache_sg",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {
						"description": "Security group for ElastiCache",
						"egress": [{"cidr_blocks": ["0.0.0.0/0"], "from_port": 0, "protocol": "-1", "to_port": 0}],
						"ingress": [
							{"cidr_blocks": ["0.0.0.0/0"], "from_port": 11211, "protocol": "tcp", "to_port": 11211},
							{"cidr_blocks": ["0.0.0.0/0"], "from_port": 3306, "protocol": "tcp", "to_port": 3306}
						],
						"name": "e2e-sg",
						"vpc_id": "vpc-123"
					},
					"after_unknown": {
						"arn": true,
						"egress": [{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []}],
						"id": true,
						"ingress": [
							{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []},
							{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []}
						],
						"owner_id": true,
						"tags_all": true
					},
					"before_sensitive": false,
					"after_sensitive": {
						"egress": [{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []}],
						"ingress": [
							{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []},
							{"cidr_blocks": [false], "ipv6_cidr_blocks": [], "prefix_list_ids": [], "security_groups": []}
						],
						"tags_all": {}
					}
				}
			}
		],
		"output_changes": {
			"pubsub_id": {
				"actions": ["create"],
				"before": null,
				"after": "",
				"after_unknown": false,
				"before_sensitive": false,
				"after_sensitive": false
			},
			"redis_endpoint": {
				"actions": ["create"],
				"before": null,
				"after_unknown": true,
				"before_sensitive": false,
				"after_sensitive": false
			}
		},
		"prior_state": {
			"format_version": "1.0",
			"terraform_version": "1.11.5",
			"values": {"root_module": {}}
		}
	}`

	result := formatTerraformPlan(raw)

	// Must be formatted, not raw JSON
	require.NotContains(t, result, `"format_version"`, "formatter returned raw JSON unchanged")

	// Version header
	require.Contains(t, result, "Terraform v1.11.5")

	// ElastiCache resource
	require.Contains(t, result, "# aws_elasticache_cluster.example_memcached will be created")
	require.Contains(t, result, `+ resource "aws_elasticache_cluster" "example_memcached"`)
	require.Contains(t, result, `"memcached"`)         // engine value
	require.Contains(t, result, "(known after apply)") // arn, id, etc.

	// Security group resource with nested arrays
	require.Contains(t, result, "# aws_security_group.elasticache_sg will be created")
	require.Contains(t, result, `+ resource "aws_security_group" "elasticache_sg"`)
	require.Contains(t, result, `"0.0.0.0/0"`) // cidr_blocks in nested array
	require.Contains(t, result, "vpc_id")

	// Outputs
	require.Contains(t, result, "─── Outputs ───")
	require.Contains(t, result, "+ pubsub_id")
	require.Contains(t, result, "+ redis_endpoint")

	// Summary
	require.Contains(t, result, "Plan: 2 to add, 0 to change, 0 to destroy.")

	// Verify multi-line human-readable format
	lines := strings.Split(result, "\n")
	require.Greater(t, len(lines), 15, "complex plan should have many readable lines")
}

func TestToStringBoolMap_NestedTypes(t *testing.T) {
	// Production after_unknown/after_sensitive can contain nested arrays and maps.
	// These should be ignored (only top-level bool=true matters for flat attribute display).
	input := map[string]interface{}{
		"arn":                        true,                                                                                                 // bool true → flagged
		"id":                         true,                                                                                                 // bool true → flagged
		"cache_nodes":                true,                                                                                                 // bool true → flagged
		"log_delivery_configuration": []interface{}{},                                                                                      // empty array → ignored
		"security_group_ids":         true,                                                                                                 // bool true → flagged
		"tags_all":                   true,                                                                                                 // bool true → flagged
		"subnet_ids":                 []interface{}{false, false, false},                                                                   // array of bools → ignored
		"egress":                     []interface{}{map[string]interface{}{"cidr_blocks": []interface{}{false}}},                           // nested object in array → ignored
		"ingress":                    []interface{}{map[string]interface{}{"cidr_blocks": []interface{}{false}}, map[string]interface{}{}}, // multiple nested → ignored
	}
	result, allFlagged := toStringBoolMap(input)
	require.False(t, allFlagged)

	// Only top-level bool=true keys should be in the result
	require.True(t, result["arn"])
	require.True(t, result["id"])
	require.True(t, result["cache_nodes"])
	require.True(t, result["security_group_ids"])
	require.True(t, result["tags_all"])

	// Nested types should NOT be in result
	require.False(t, result["log_delivery_configuration"])
	require.False(t, result["subnet_ids"])
	require.False(t, result["egress"])
	require.False(t, result["ingress"])
}

func TestFormatTerraformPlan_NestedArrayValues(t *testing.T) {
	// Test that complex nested values (arrays of objects) in "after" are rendered
	// inline by formatValue, not dropped or causing errors.
	raw := `{
		"format_version": "1.2",
		"terraform_version": "1.11.5",
		"resource_changes": [{
			"address": "aws_security_group.sg",
			"type": "aws_security_group",
			"name": "sg",
			"change": {
				"actions": ["create"],
				"before": null,
				"after": {
					"name": "test-sg",
					"egress": [{"cidr_blocks": ["0.0.0.0/0"], "from_port": 0, "protocol": "-1", "to_port": 0}],
					"ingress": [
						{"cidr_blocks": ["10.0.0.0/8"], "from_port": 443, "protocol": "tcp", "to_port": 443}
					]
				},
				"after_unknown": {
					"id": true,
					"egress": [{"cidr_blocks": [false]}],
					"ingress": [{"cidr_blocks": [false]}]
				},
				"before_sensitive": false,
				"after_sensitive": {}
			}
		}]
	}`

	result := formatTerraformPlan(raw)

	require.Contains(t, result, "# aws_security_group.sg will be created")
	// Nested array values should be formatted inline
	require.Contains(t, result, "egress")
	require.Contains(t, result, "ingress")
	require.Contains(t, result, "0.0.0.0/0")
	require.Contains(t, result, "10.0.0.0/8")
	// id should be marked as unknown
	require.Contains(t, result, "(known after apply)")
}
