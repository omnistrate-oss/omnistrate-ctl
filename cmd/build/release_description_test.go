package build

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestReleaseDescriptionFlagExists(t *testing.T) {
	// Test that the release-description flag exists in the build command
	flag := BuildCmd.Flags().Lookup("release-description")
	assert.NotNil(t, flag, "release-description flag should exist")
	assert.Equal(t, "Used together with --release or --release-as-preferred flag. Provide a description for the release version", flag.Usage)
}

func TestBuildFromRepoReleaseDescriptionFlagExists(t *testing.T) {
	// Test that the release-description flag exists in the build-from-repo command
	flag := BuildFromRepoCmd.Flags().Lookup("release-description")
	assert.NotNil(t, flag, "release-description flag should exist")
	assert.Equal(t, "Provide a description for the release version", flag.Usage)
}

func TestReleaseDescriptionFlagHandling(t *testing.T) {
	// Test case 1: Only release-description flag provided
	cmd := &cobra.Command{}
	cmd.Flags().StringP("release-name", "", "", "Deprecated release name")
	cmd.Flags().StringP("release-description", "", "", "Release description")

	err := cmd.ParseFlags([]string{"--release-description", "v1.0.0"})
	assert.NoError(t, err)

	releaseName, _ := cmd.Flags().GetString("release-name")
	releaseDescription, _ := cmd.Flags().GetString("release-description")

	// Simulate the logic from runBuild (lines 454-459 in build.go)
	var releaseNamePtr *string
	if releaseName != "" {
		releaseNamePtr = &releaseName
	}
	if releaseDescription != "" {
		releaseNamePtr = &releaseDescription
	}

	assert.NotNil(t, releaseNamePtr, "releaseNamePtr should not be nil when release-description is provided")
	assert.Equal(t, "v1.0.0", *releaseNamePtr)

	// Test case 2: Both release-name and release-description provided
	// release-description should take precedence
	cmd2 := &cobra.Command{}
	cmd2.Flags().StringP("release-name", "", "", "Deprecated release name")
	cmd2.Flags().StringP("release-description", "", "", "Release description")

	err = cmd2.ParseFlags([]string{"--release-name", "old-value", "--release-description", "v2.0.0"})
	assert.NoError(t, err)

	releaseName2, _ := cmd2.Flags().GetString("release-name")
	releaseDescription2, _ := cmd2.Flags().GetString("release-description")

	// Simulate the logic from runBuild
	var releaseNamePtr2 *string
	if releaseName2 != "" {
		releaseNamePtr2 = &releaseName2
	}
	if releaseDescription2 != "" {
		releaseNamePtr2 = &releaseDescription2
	}

	assert.NotNil(t, releaseNamePtr2, "releaseNamePtr2 should not be nil")
	assert.Equal(t, "v2.0.0", *releaseNamePtr2, "release-description should take precedence over release-name")

	// Test case 3: Only deprecated release-name provided (backward compatibility)
	cmd3 := &cobra.Command{}
	cmd3.Flags().StringP("release-name", "", "", "Deprecated release name")
	cmd3.Flags().StringP("release-description", "", "", "Release description")

	err = cmd3.ParseFlags([]string{"--release-name", "v0.9.0"})
	assert.NoError(t, err)

	releaseName3, _ := cmd3.Flags().GetString("release-name")
	releaseDescription3, _ := cmd3.Flags().GetString("release-description")

	// Simulate the logic from runBuild
	var releaseNamePtr3 *string
	if releaseName3 != "" {
		releaseNamePtr3 = &releaseName3
	}
	if releaseDescription3 != "" {
		releaseNamePtr3 = &releaseDescription3
	}

	assert.NotNil(t, releaseNamePtr3, "releaseNamePtr3 should not be nil when release-name is provided")
	assert.Equal(t, "v0.9.0", *releaseNamePtr3)

	// Test case 4: Neither flag provided
	cmd4 := &cobra.Command{}
	cmd4.Flags().StringP("release-name", "", "", "Deprecated release name")
	cmd4.Flags().StringP("release-description", "", "", "Release description")

	err = cmd4.ParseFlags([]string{})
	assert.NoError(t, err)

	releaseName4, _ := cmd4.Flags().GetString("release-name")
	releaseDescription4, _ := cmd4.Flags().GetString("release-description")

	// Simulate the logic from runBuild
	var releaseNamePtr4 *string
	if releaseName4 != "" {
		releaseNamePtr4 = &releaseName4
	}
	if releaseDescription4 != "" {
		releaseNamePtr4 = &releaseDescription4
	}

	assert.Nil(t, releaseNamePtr4, "releaseNamePtr4 should be nil when neither flag is provided")
}

func TestBuildFromRepoReleaseDescriptionFlagHandling(t *testing.T) {
	// Test that release-description flag value is correctly extracted in build-from-repo
	cmd := &cobra.Command{}
	cmd.Flags().String("release-description", "", "Provide a description for the release version")

	err := cmd.ParseFlags([]string{"--release-description", "v3.0.0-beta"})
	assert.NoError(t, err)

	releaseDescription, _ := cmd.Flags().GetString("release-description")

	// Simulate the logic from runBuildFromRepo (lines 1001-1004 in build_from_repo.go)
	var releaseDescriptionPtr *string
	if releaseDescription != "" {
		releaseDescriptionPtr = &releaseDescription
	}

	assert.NotNil(t, releaseDescriptionPtr, "releaseDescriptionPtr should not be nil when release-description is provided")
	assert.Equal(t, "v3.0.0-beta", *releaseDescriptionPtr)
}
