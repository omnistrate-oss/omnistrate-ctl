package deploymentcell

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRestartCommand(t *testing.T) {
	command, _, err := Cmd.Find([]string{"restart"})
	require.NoError(t, err)
	require.Same(t, restartCmd, command)
	require.Equal(t, "restart [deployment-cell-id]", command.Use)
	require.NotEmpty(t, command.Short)
	require.Contains(t, command.Long, "restart has been requested")
	require.Contains(t, command.Example, "omnistrate-ctl deployment-cell restart hc-")
	require.NotNil(t, command.RunE)
	require.True(t, command.SilenceUsage)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "one ID", args: []string{"hc-12345678"}},
		{name: "missing ID", want: "accepts 1 arg(s), received 0"},
		{name: "extra ID", args: []string{"hc-12345678", "hc-87654321"}, want: "accepts 1 arg(s), received 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := command.ValidateArgs(tc.args)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRestartRejectsInvalidInputBeforeAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name, id, output, want string
	}{
		{name: "empty ID", output: "table", want: "deployment cell ID is required"},
		{name: "blank ID", id: " \t", output: "json", want: "deployment cell ID is required"},
		{name: "unsupported output", id: "hc-12345678", output: "yaml", want: "unsupported output format: yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := &cobra.Command{Use: restartCmd.Use}
			command.Flags().String(common.OutputFlag, tc.output, "Output format")
			require.ErrorContains(t, runRestart(command, []string{tc.id}), tc.want)
		})
	}
}
