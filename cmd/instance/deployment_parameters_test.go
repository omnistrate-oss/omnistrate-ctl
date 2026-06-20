package instance

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestFindMatchingResource(t *testing.T) {
	require := require.New(t)

	records := []openapiclientfleet.ResourceSearchRecord{
		{
			Id:                     "res-dev",
			Name:                   "mySQL",
			ServiceName:            "mysql",
			ProductTierName:        "mysql",
			ServiceId:              "svc-1",
			ProductTierId:          "pt-dev",
			ServiceEnvironmentId:   "env-dev",
			ServiceEnvironmentName: "Dev",
		},
		{
			Id:                     "res-prod",
			Name:                   "mySQL",
			ServiceName:            "mysql",
			ProductTierName:        "mysql",
			ServiceId:              "svc-1",
			ProductTierId:          "pt-prod",
			ServiceEnvironmentId:   "env-prod",
			ServiceEnvironmentName: "Production",
		},
	}

	// Match by environment name (case-insensitive)
	match, err := findMatchingResource(records, "mysql", "mysql", "mySQL", "production")
	require.NoError(err)
	require.Equal("res-prod", match.Id)
	require.Equal("pt-prod", match.ProductTierId)

	match, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "Dev")
	require.NoError(err)
	require.Equal("res-dev", match.Id)

	// Match by environment ID
	match, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "env-prod")
	require.NoError(err)
	require.Equal("res-prod", match.Id)

	// No environment match -> not found
	_, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "staging")
	require.Error(err)
	require.Contains(err.Error(), "not found")
}

func TestTemplateValueForParam(t *testing.T) {
	require := require.New(t)

	// String with no default -> empty string placeholder
	require.Equal("", templateValueForParam(ParameterInfo{Key: "name", Type: "String"}))

	// String with default -> the default
	require.Equal("default", templateValueForParam(ParameterInfo{
		Key: "databaseName", Type: "String", DefaultValue: utils.ToPtr("default"),
	}))

	// Bool with no default -> false; with default -> parsed bool
	require.Equal(false, templateValueForParam(ParameterInfo{Key: "tls", Type: "Boolean"}))
	require.Equal(true, templateValueForParam(ParameterInfo{
		Key: "tls", Type: "Boolean", DefaultValue: utils.ToPtr("true"),
	}))

	// Number with no default -> 0; with default -> parsed float
	require.Equal(0, templateValueForParam(ParameterInfo{Key: "size", Type: "Float64"}))
	require.Equal(float64(2.5), templateValueForParam(ParameterInfo{
		Key: "size", Type: "Float64", DefaultValue: utils.ToPtr("2.5"),
	}))

	// Integer with default -> parsed int64
	require.Equal(int64(8080), templateValueForParam(ParameterInfo{
		Key: "port", Type: "Integer", DefaultValue: utils.ToPtr("8080"),
	}))

	// List with no default -> empty slice
	require.Equal([]any{}, templateValueForParam(ParameterInfo{Key: "zones", Type: "String", IsList: true}))

	// List with CSV default -> string slice
	require.Equal([]string{"a", "b"}, templateValueForParam(ParameterInfo{
		Key: "zones", Type: "String", IsList: true, DefaultValue: utils.ToPtr("a,b"),
	}))

	// Malformed numeric default -> falls back to raw string
	require.Equal("not-a-number", templateValueForParam(ParameterInfo{
		Key: "size", Type: "Float64", DefaultValue: utils.ToPtr("not-a-number"),
	}))
}

func TestBuildParamTemplate(t *testing.T) {
	require := require.New(t)

	params := []ParameterInfo{
		{Key: "databaseName", Type: "String", DefaultValue: utils.ToPtr("default")},
		{Key: "password", Type: "Password"},
		{Key: "port", Type: "Integer", DefaultValue: utils.ToPtr("3306")},
	}

	template := buildParamTemplate(params)

	require.Len(template, 3)
	require.Equal("default", template["databaseName"])
	require.Equal("", template["password"])
	require.Equal(int64(3306), template["port"])
}

func TestDeploymentParametersTemplateCommand(t *testing.T) {
	require := require.New(t)

	// template subcommand is registered under deployment-parameters
	var found *cobra.Command
	for _, c := range deploymentParametersCmd.Commands() {
		if c.Name() == "template" {
			found = c
			break
		}
	}
	require.NotNil(found, "template subcommand should be registered")

	// required flags exist on the template subcommand
	require.NotNil(found.Flag("service"))
	require.NotNil(found.Flag("plan"))
	require.NotNil(found.Flag("resource"))
	require.NotNil(found.Flag("environment"))
	require.NotNil(found.Flag("version"))

	// parent command has the environment flag (from Task 1)
	require.NotNil(deploymentParametersCmd.Flag("environment"))
}
