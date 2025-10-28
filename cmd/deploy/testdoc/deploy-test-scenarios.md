# Deploy Command Test Scenarios Matrix

## Test Scenario Categories

### 1. Authentication & Authorization
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| AUTH-001 | User not logged in | `omctl deploy spec.yaml` | Error: "not logged in. Please run 'omctl login' to authenticate" |
| AUTH-002 | Invalid/expired token | `omctl deploy spec.yaml` | Token refresh or login prompt |
| AUTH-003 | Valid authenticated user | `omctl deploy spec.yaml` | Proceeds with deployment |

### 2. Spec File Validation
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| SPEC-001 | No spec file provided, no compose.yaml | `omctl deploy` | Auto-detect repo build mode |
| SPEC-002 | Invalid spec file path | `omctl deploy nonexistent.yaml` | Error: "spec file does not exist" |
| SPEC-003 | Valid docker-compose.yaml | `omctl deploy docker-compose.yaml` | Docker compose spec type detected |
| SPEC-004 | Valid service plan spec | `omctl deploy serviceplan.yaml` | Service plan spec type detected |
| SPEC-005 | YAML with template expressions | `omctl deploy spec-with-templates.yaml` | Template expressions processed correctly |
| SPEC-006 | Spec without x-omnistrate keys | `omctl deploy plain-docker-compose.yaml` | Warning about missing omnistrate configurations |
| SPEC-007 | Malformed YAML file | `omctl deploy malformed.yaml` | Error parsing YAML |
| SPEC-008 | Empty spec file | `omctl deploy empty.yaml` | Error: invalid or empty spec |

### 3. Deployment Type Validation
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| DTYPE-001 | Valid hosted deployment | `omctl deploy spec.yaml --deployment-type hosted` | Deployment type set to hosted |
| DTYPE-002 | Valid BYOA deployment | `omctl deploy spec.yaml --deployment-type byoa` | Deployment type set to byoa |
| DTYPE-003 | Invalid deployment type | `omctl deploy spec.yaml --deployment-type invalid` | Error: "invalid deployment-type. Valid values are: hosted, byoa" |
| DTYPE-004 | Deployment type in spec overrides flag | `omctl deploy byoa-spec.yaml --deployment-type hosted` | Spec deployment type takes precedence |
| DTYPE-005 | No deployment type specified | `omctl deploy spec.yaml` | Defaults to hosted |

### 4. Cloud Provider Account Management
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| CLOUD-001 | No cloud accounts linked | `omctl deploy spec.yaml` | Error: "no cloud provider accounts found" |
| CLOUD-002 | Only non-READY accounts | `omctl deploy spec.yaml` | Error: "no READY accounts found" |
| CLOUD-003 | Multiple READY accounts | `omctl deploy spec.yaml` | Uses first available READY account |
| CLOUD-004 | AWS account in spec file | `omctl deploy aws-spec.yaml` | Uses specified AWS account |
| CLOUD-005 | GCP account in spec file | `omctl deploy gcp-spec.yaml` | Uses specified GCP account |
| CLOUD-006 | Azure account in spec file | `omctl deploy azure-spec.yaml` | Uses specified Azure account |
| CLOUD-007 | Account specified but not linked | `omctl deploy spec-with-unlinked-account.yaml` | Error: "account not linked" |
| CLOUD-008 | Account linked but not READY | `omctl deploy spec-with-pending-account.yaml` | Error: "account not READY" |

### 5. Service Name Handling
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| SVC-001 | Custom service name provided | `omctl deploy spec.yaml --product-name my-service` | Uses "my-service" |
| SVC-002 | No service name, use directory | `omctl deploy spec.yaml` | Uses sanitized directory name |
| SVC-003 | Invalid service name characters | `omctl deploy spec.yaml --product-name "My Service!"` | Sanitizes to valid name |
| SVC-004 | Service name starting with numbers | `omctl deploy spec.yaml --product-name "123service"` | Prefixed with "svc-" |
| SVC-005 | Very long service name | `omctl deploy spec.yaml --product-name "very-long-service-name..."` | Truncated appropriately |

### 6. Environment Management
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| ENV-001 | Default environment | `omctl deploy spec.yaml` | Uses "Prod" environment with "prod" type |
| ENV-002 | Custom environment | `omctl deploy spec.yaml --environment MyEnv --environment-type staging` | Creates/uses "MyEnv" staging environment |
| ENV-003 | Environment without type | `omctl deploy spec.yaml --environment MyEnv` | Error: environment and environment-type must be used together |
| ENV-004 | Type without environment | `omctl deploy spec.yaml --environment-type dev` | Error: environment and environment-type must be used together |
| ENV-005 | Invalid environment type | `omctl deploy spec.yaml --environment Test --environment-type invalid` | Validation error for environment type |

### 7. Dry Run Mode
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| DRY-001 | Dry run with valid spec | `omctl deploy spec.yaml --dry-run` | Validation only, no actual deployment |
| DRY-002 | Dry run with invalid spec | `omctl deploy invalid.yaml --dry-run` | Shows validation errors |
| DRY-003 | Dry run with auth issues | `omctl deploy spec.yaml --dry-run` (not logged in) | Shows auth error |
| DRY-004 | Dry run with missing accounts | `omctl deploy spec.yaml --dry-run` | Shows account validation errors |

### 8. Instance Management
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| INST-001 | No existing instances | `omctl deploy spec.yaml` | Creates new instance automatically |
| INST-002 | One existing instance | `omctl deploy spec.yaml` | Upgrades existing instance |
| INST-003 | Multiple existing instances | `omctl deploy spec.yaml` | Prompts user to select instance or create new |
| INST-004 | Specific instance ID | `omctl deploy spec.yaml --instance-id abc123` | Upgrades specified instance |
| INST-005 | Invalid instance ID | `omctl deploy spec.yaml --instance-id invalid` | Creates new instance |
| INST-006 | Instance creation with params | `omctl deploy spec.yaml --param '{"key":"value"}'` | Creates instance with parameters |
| INST-007 | Instance with param file | `omctl deploy spec.yaml --param-file params.json` | Creates instance with file parameters |
| INST-008 | Missing required parameters | `omctl deploy spec.yaml` | Error: missing required parameters |

### 9. Resource Management
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| RES-001 | Single resource in spec | `omctl deploy spec.yaml` | Uses the single resource |
| RES-002 | Multiple resources in spec | `omctl deploy spec.yaml` | Prompts user to select resource |
| RES-003 | Specific resource ID | `omctl deploy spec.yaml --resource-id res123` | Uses specified resource |
| RES-004 | Invalid resource ID | `omctl deploy spec.yaml --resource-id invalid` | Error: resource not found |
| RES-005 | All internal resources | `omctl deploy internal-spec.yaml` | Creates passive cluster resource |
| RES-006 | Mixed internal/external resources | `omctl deploy mixed-spec.yaml` | Processes resources appropriately |

### 10. Cloud Provider Specific Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| CP-001 | AWS with all parameters | `omctl deploy spec.yaml --cloud-provider aws --region us-east-1` | Creates AWS instance |
| CP-002 | GCP with all parameters | `omctl deploy spec.yaml --cloud-provider gcp --region us-central1` | Creates GCP instance |
| CP-003 | Azure with all parameters | `omctl deploy spec.yaml --cloud-provider azure --region eastus` | Creates Azure instance |
| CP-004 | Invalid cloud provider | `omctl deploy spec.yaml --cloud-provider invalid` | Error: unsupported cloud provider |
| CP-005 | Invalid region for provider | `omctl deploy spec.yaml --cloud-provider aws --region invalid-region` | Error: unsupported region |
| CP-006 | Region without provider | `omctl deploy spec.yaml --region us-east-1` | Auto-detects or uses default provider |

### 11. BYOA Specific Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| BYOA-001 | BYOA without cloud account instances | `omctl deploy byoa-spec.yaml` | Creates cloud account instances first |
| BYOA-002 | BYOA with existing cloud account | `omctl deploy byoa-spec.yaml` | Uses existing cloud account instance |
| BYOA-003 | BYOA cloud account creation flow | `omctl deploy byoa-spec.yaml` | Prompts for cloud credentials |
| BYOA-004 | BYOA account verification timeout | `omctl deploy byoa-spec.yaml` | Error after verification timeout |
| BYOA-005 | BYOA account verification success | `omctl deploy byoa-spec.yaml` | Proceeds after account verification |

### 12. Multi-Resource Scenarios
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| MULTI-001 | Multiple services, all internal | `omctl deploy multi-internal.yaml` | Injects Cluster passive resource |
| MULTI-002 | Multiple services, mixed mode | `omctl deploy multi-mixed.yaml` | Sets appropriate internal flags |
| MULTI-003 | Multiple services with dependencies | `omctl deploy multi-deps.yaml` | Preserves dependency order |
| MULTI-004 | Services with parameter mapping | `omctl deploy multi-params.yaml` | Creates parameter dependency maps |

### 13. Build Integration Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| BUILD-001 | Docker compose with Docker build | `omctl deploy compose.yaml` | Builds and pushes Docker images |
| BUILD-002 | Skip Docker build | `omctl deploy compose.yaml --skip-docker-build` | Uses existing images |
| BUILD-003 | Multiple platform build | `omctl deploy compose.yaml --platforms linux/amd64 --platforms linux/arm64` | Builds for multiple platforms |
| BUILD-004 | Build with custom platforms | `omctl deploy compose.yaml --platforms linux/arm64` | Builds for ARM64 only |

### 14. Error Handling & Edge Cases
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| ERR-001 | Network connectivity issues | `omctl deploy spec.yaml` | Graceful error handling |
| ERR-002 | API rate limiting | `omctl deploy spec.yaml` | Retry with backoff |
| ERR-003 | Large spec file processing | `omctl deploy large-spec.yaml` | Handles large files efficiently |
| ERR-004 | Spec with circular dependencies | `omctl deploy circular-deps.yaml` | Error: circular dependency detected |
| ERR-005 | Interrupted deployment | `Ctrl+C during deploy` | Graceful cleanup |
| ERR-006 | Disk space issues | `omctl deploy spec.yaml` (low disk) | Error: insufficient disk space |

### 15. Integration & End-to-End Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| E2E-001 | Complete new service deployment | `omctl deploy new-service.yaml` | Service created and instance deployed |
| E2E-002 | Service update deployment | `omctl deploy updated-service.yaml` | Service updated and instance upgraded |
| E2E-003 | Multi-environment deployment | `omctl deploy spec.yaml --environment staging --environment-type staging` | Deploys to staging environment |
| E2E-004 | Cross-cloud deployment | `omctl deploy multi-cloud-spec.yaml` | Handles multiple cloud providers |
| E2E-005 | Complete workflow with monitoring | `omctl deploy spec.yaml` | Shows deployment progress and completion |

### 16. Parameter Validation Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| PARAM-001 | Valid JSON parameters | `omctl deploy spec.yaml --param '{"key":"value"}'` | Parameters parsed correctly |
| PARAM-002 | Invalid JSON parameters | `omctl deploy spec.yaml --param 'invalid-json'` | Error: invalid JSON format |
| PARAM-003 | Parameters from file | `omctl deploy spec.yaml --param-file params.json` | File parameters loaded |
| PARAM-004 | Missing parameter file | `omctl deploy spec.yaml --param-file missing.json` | Error: file not found |
| PARAM-005 | Both param and param-file | `omctl deploy spec.yaml --param '{}' --param-file params.json` | File parameters take precedence |
| PARAM-006 | Empty parameters | `omctl deploy spec.yaml --param '{}'` | Uses default parameter values |

### 17. Template Processing Tests
| Test ID | Scenario | Command | Expected Result |
|---------|----------|---------|----------------|
| TMPL-001 | Simple file inclusion | `omctl deploy spec-with-file-ref.yaml` | File content included |
| TMPL-002 | Nested file inclusions | `omctl deploy spec-with-nested-refs.yaml` | All nested files processed |
| TMPL-003 | Missing template file | `omctl deploy spec-with-missing-ref.yaml` | Error: template file not found |
| TMPL-004 | Circular template references | `omctl deploy spec-with-circular-refs.yaml` | Error: circular reference |
| TMPL-005 | Template with indentation | `omctl deploy spec-with-indented-ref.yaml` | Proper indentation preserved |

## Test Data Requirements

### Sample Spec Files Needed:
1. `minimal-docker-compose.yaml` - Basic Docker Compose spec
2. `service-plan-spec.yaml` - Service plan with x-omnistrate configurations
3. `byoa-spec.yaml` - BYOA deployment configuration
4. `hosted-spec.yaml` - Hosted deployment configuration
5. `multi-resource-spec.yaml` - Multiple services/resources
6. `template-spec.yaml` - File with template expressions
7. `aws-spec.yaml` - AWS-specific configurations
8. `gcp-spec.yaml` - GCP-specific configurations
9. `azure-spec.yaml` - Azure-specific configurations
10. `invalid-spec.yaml` - Malformed YAML for error testing

### Parameter Files Needed:
1. `basic-params.json` - Basic instance parameters
2. `aws-params.json` - AWS-specific parameters
3. `gcp-params.json` - GCP-specific parameters
4. `azure-params.json` - Azure-specific parameters
5. `invalid-params.json` - Invalid JSON for error testing

### Environment Setup Requirements:
1. Test accounts for AWS, GCP, Azure (in various states: READY, NOT_READY)
2. Test services with different configurations
3. Test instances in various states
4. Docker environment for build testing
5. Git repository with sample code for build-from-repo testing

## Automation Recommendations

### Priority 1 (Critical Path):
- AUTH-001, AUTH-003
- SPEC-003, SPEC-004
- DTYPE-001, DTYPE-002, DTYPE-003
- CLOUD-002, CLOUD-003
- SVC-001, SVC-002
- INST-001, INST-002
- E2E-001

### Priority 2 (Core Features):
- All cloud provider specific tests (CP-*)
- BYOA workflow tests (BYOA-*)
- Multi-resource scenarios (MULTI-*)
- Parameter validation (PARAM-*)

### Priority 3 (Edge Cases):
- Error handling tests (ERR-*)
- Template processing (TMPL-*)
- Advanced scenarios

This comprehensive test matrix covers all major deployment scenarios and edge cases for the omnistrate-ctl deploy command.