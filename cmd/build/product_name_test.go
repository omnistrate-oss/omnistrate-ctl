package build

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestProductNameFlagExists(t *testing.T) {
	// Test that the product-name flag exists in the build command
	flag := BuildCmd.Flags().Lookup("product-name")
	assert.NotNil(t, flag, "product-name flag should exist")
	assert.Equal(t, "Name of the service. A service can have multiple service plans. The build command will build a new or existing service plan inside the specified service.", flag.Usage)
}

func TestBuildFromRepoProductNameFlagExists(t *testing.T) {
	// Test that the product-name flag exists in the build-from-repo command
	flag := BuildFromRepoCmd.Flags().Lookup("product-name")
	assert.NotNil(t, flag, "product-name flag should exist")
	assert.Equal(t, "Specify a custom service name. If not provided, the repository name will be used.", flag.Usage)
}

func TestProductNameAndNameFlagHandling(t *testing.T) {
	// Test case 1: Only name flag provided (deprecated but still works)
	cmd := &cobra.Command{}
	cmd.Flags().StringP("name", "n", "", "Name of the service")
	cmd.Flags().StringP("product-name", "", "", "Name of the service")

	err := cmd.ParseFlags([]string{"--name", "test-service"})
	assert.NoError(t, err)

	name, _ := cmd.Flags().GetString("name")
	productName, _ := cmd.Flags().GetString("product-name")

	// Simulate the logic from runBuild
	finalName := name
	if productName != "" {
		finalName = productName
	}

	assert.Equal(t, "test-service", finalName)

	// Test case 2: Only product-name flag provided
	cmd2 := &cobra.Command{}
	cmd2.Flags().StringP("name", "n", "", "Name of the service")
	cmd2.Flags().StringP("product-name", "", "", "Name of the service")

	err = cmd2.ParseFlags([]string{"--product-name", "product-service"})
	assert.NoError(t, err)

	name2, _ := cmd2.Flags().GetString("name")
	productName2, _ := cmd2.Flags().GetString("product-name")

	// Simulate the logic from runBuild
	finalName2 := name2
	if productName2 != "" {
		finalName2 = productName2
	}

	assert.Equal(t, "product-service", finalName2)

	// Test case 3: Test that deprecated name flag still works
	cmd3 := &cobra.Command{}
	cmd3.Flags().StringP("name", "n", "", "Name of the service")
	cmd3.Flags().StringP("product-name", "", "", "Name of the service")
	err = cmd3.Flags().MarkDeprecated("name", "use --product-name instead")
	assert.NoError(t, err)

	err = cmd3.ParseFlags([]string{"--name", "deprecated-service"})
	assert.NoError(t, err)

	name3, _ := cmd3.Flags().GetString("name")
	productName3, _ := cmd3.Flags().GetString("product-name")

	// Simulate the logic from runBuild
	finalName3 := name3
	if productName3 != "" {
		finalName3 = productName3
	}

	assert.Equal(t, "deprecated-service", finalName3)
}
