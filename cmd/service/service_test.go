package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServicePlanNestedCommandRegistered(t *testing.T) {
	planCmd, _, err := Cmd.Find([]string{"plan"})

	require.NoError(t, err)
	require.NotNil(t, planCmd)
	require.Equal(t, "plan", planCmd.Name())

	expected := []string{
		"delete",
		"describe",
		"describe-version",
		"disable-feature",
		"enable-feature",
		"list",
		"list-versions",
		"release",
		"set-default",
		"update",
	}
	actual := make([]string, 0, len(planCmd.Commands()))
	for _, command := range planCmd.Commands() {
		actual = append(actual, command.Name())
	}
	require.ElementsMatch(t, expected, actual)
}
