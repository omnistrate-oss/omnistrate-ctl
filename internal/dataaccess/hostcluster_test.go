package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostClusterNodepoolEntityType(t *testing.T) {
	t.Run("supported providers", func(t *testing.T) {
		tests := []struct {
			cloudProvider string
			entityType    string
		}{
			{cloudProvider: "aws", entityType: "NODE_GROUP"},
			{cloudProvider: "gcp", entityType: "NODEPOOL"},
			{cloudProvider: "azure", entityType: "AZURE_NODEPOOL"},
		}

		for _, tt := range tests {
			entityType, err := hostClusterNodepoolEntityType(tt.cloudProvider)
			require.NoError(t, err)
			assert.Equal(t, tt.entityType, entityType)
		}
	})

	t.Run("nebius unsupported", func(t *testing.T) {
		entityType, err := hostClusterNodepoolEntityType("nebius")
		require.Error(t, err)
		assert.Empty(t, entityType)
		assert.Contains(t, err.Error(), "Nebius deployment cells is not yet supported")
	})
}
