package deploymentcell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_deployment_cell_config_template_workload_identities(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	tmpDir := t.TempDir()
	currentTemplatePath := filepath.Join(tmpDir, "current-template.yaml")
	updatedTemplatePath := filepath.Join(tmpDir, "updated-template.yaml")
	restoreTemplatePath := filepath.Join(tmpDir, "restore-template.yaml")

	cmd.RootCmd.SetArgs([]string{
		"deployment-cell", "describe-config-template",
		"--environment", "GLOBAL",
		"--cloud", "aws",
		"--output-file", currentTemplatePath,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	currentTemplate := readDeploymentCellTemplateForSmokeTest(t, currentTemplatePath)
	restoreTemplate := currentTemplate
	if restoreTemplate.WorkloadIdentities == nil {
		restoreTemplate.WorkloadIdentities = []model.ManagedWorkloadIdentity{}
	}
	writeDeploymentCellTemplateForSmokeTest(t, restoreTemplatePath, restoreTemplate)
	defer func() {
		cmd.RootCmd.SetArgs([]string{
			"deployment-cell", "update-config-template",
			"--environment", "GLOBAL",
			"--cloud", "aws",
			"--file", restoreTemplatePath,
		})
		_ = cmd.RootCmd.ExecuteContext(ctx)
	}()

	description := "Smoke test workload identity"
	currentTemplate.WorkloadIdentities = []model.ManagedWorkloadIdentity{
		{
			Identifier:  "ctl-smoke-queue-writer",
			Description: &description,
			Bindings: []model.ManagedWorkloadIdentityBinding{
				{
					ServiceAccount: &model.ManagedWorkloadIdentityServiceAccount{
						Namespace: "queue-system",
						Name:      "queue-writer",
					},
				},
			},
			Permissions: &model.ManagedWorkloadIdentityPermissions{
				Policies: map[string]string{
					"aws": `{"Statement":[{"Action":["sqs:SendMessage"],"Effect":"Allow","Resource":"*"}]}`,
				},
			},
		},
	}
	writeDeploymentCellTemplateForSmokeTest(t, updatedTemplatePath, currentTemplate)

	cmd.RootCmd.SetArgs([]string{
		"deployment-cell", "update-config-template",
		"--environment", "GLOBAL",
		"--cloud", "aws",
		"--file", updatedTemplatePath,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{
		"deployment-cell", "describe-config-template",
		"--environment", "GLOBAL",
		"--cloud", "aws",
		"--output-file", currentTemplatePath,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	updatedTemplate := readDeploymentCellTemplateForSmokeTest(t, currentTemplatePath)
	require.Len(updatedTemplate.WorkloadIdentities, 1)
	require.Equal("ctl-smoke-queue-writer", updatedTemplate.WorkloadIdentities[0].Identifier)
}

func readDeploymentCellTemplateForSmokeTest(t *testing.T, path string) model.DeploymentCellTemplate {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var template model.DeploymentCellTemplate
	require.NoError(t, yaml.Unmarshal(data, &template))
	return template
}

func writeDeploymentCellTemplateForSmokeTest(t *testing.T, path string, template model.DeploymentCellTemplate) {
	t.Helper()

	data, err := yaml.Marshal(template)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
}
