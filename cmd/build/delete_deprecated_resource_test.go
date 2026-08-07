package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDeleteDeprecatedResourceCommandRemoved(t *testing.T) {
	for _, command := range BuildCmd.Commands() {
		require.NotEqual(t, "delete-deprecated-resource", command.Name())
	}
}
