package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountCommandIncludesCloudNativeNetwork(t *testing.T) {
	require.NotNil(t, Cmd)

	subCmds := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	assert.True(t, subCmds["cloud-native-network"], "expected cloud-native-network subcommand")
}
