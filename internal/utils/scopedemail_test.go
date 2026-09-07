package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatEmailWithScopedOrg(t *testing.T) {
	scoped, err := FormatEmailWithScopedOrg("customer@example.com", "org-abc123")
	require.NoError(t, err)
	assert.Equal(t, "customer+org-abc123@example.com", scoped)
}

func TestFormatEmailWithScopedOrgPreservesExistingPlusTag(t *testing.T) {
	scoped, err := FormatEmailWithScopedOrg("customer+team@example.com", "org-abc123")
	require.NoError(t, err)
	assert.Equal(t, "customer+team+org-abc123@example.com", scoped)
}

func TestFormatEmailWithScopedOrgRejectsAlreadyScopedEmail(t *testing.T) {
	_, err := FormatEmailWithScopedOrg("customer+org-abc123@example.com", "org-abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already scoped")
}

func TestFormatEmailWithScopedOrgRejectsBadInput(t *testing.T) {
	for name, tc := range map[string]struct{ email, orgID string }{
		"no at sign":    {"customer.example.com", "org-abc123"},
		"two at signs":  {"customer@example@com", "org-abc123"},
		"empty org":     {"customer@example.com", ""},
		"org bad shape": {"customer@example.com", "abc123"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := FormatEmailWithScopedOrg(tc.email, tc.orgID)
			require.Error(t, err)
		})
	}
}

func TestEmailHasScopedOrg(t *testing.T) {
	assert.True(t, EmailHasScopedOrg("customer+org-abc123@example.com"))
	assert.True(t, EmailHasScopedOrg("customer+team+org-abc123@example.com"))
	assert.False(t, EmailHasScopedOrg("customer@example.com"))
	assert.False(t, EmailHasScopedOrg("customer+team@example.com"))
	assert.False(t, EmailHasScopedOrg("not-an-email"))
}

func TestIsProductionEnvironmentType(t *testing.T) {
	assert.True(t, IsProductionEnvironmentType("PROD"))
	assert.True(t, IsProductionEnvironmentType("production"))
	assert.True(t, IsProductionEnvironmentType("  Prod  "))
	assert.False(t, IsProductionEnvironmentType("DEV"))
	assert.False(t, IsProductionEnvironmentType(""))
}
