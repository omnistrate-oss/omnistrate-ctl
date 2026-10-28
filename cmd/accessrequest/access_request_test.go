package accessrequest

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAccessRequestCommands(t *testing.T) {
	require.Equal(t, "access-request [operation] [flags]", Cmd.Use)
	require.Equal(t, "Manage deployment-cell access requests", Cmd.Short)
	require.Contains(t, Cmd.Long, "approve, deny, and revoke")

	expectedCommands := []string{"approve", "deny", "describe", "list", "revoke"}
	require.Len(t, Cmd.Commands(), len(expectedCommands))
	for index, expected := range expectedCommands {
		require.Equal(t, expected, Cmd.Commands()[index].Name())
		command, _, err := Cmd.Find([]string{expected})
		require.NoError(t, err)
		require.Equal(t, expected, command.Name())
		require.NotNil(t, command.RunE)
		require.True(t, command.SilenceUsage)
	}
	require.ErrorContains(t, Cmd.ValidateArgs([]string{"create"}), "unknown command")
}

func TestAccessRequestCommandFlags(t *testing.T) {
	require.Equal(t, "[]", listCmd.Flags().Lookup("status").DefValue)
	require.Equal(t, "false", listCmd.Flags().Lookup("include-deleted").DefValue)
	require.Equal(t, "0", approveCmd.Flags().Lookup("duration-minutes").DefValue)
}

func TestAccessRequestCommandArguments(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		command *cobra.Command
		args    []string
		wantErr string
	}{
		{name: "describe accepts request", command: describeCmd, args: []string{"ar-12345678"}},
		{name: "list rejects arguments", command: listCmd, args: []string{"extra"}, wantErr: "unknown command"},
		{name: "approve accepts request", command: approveCmd, args: []string{"ar-12345678"}},
		{name: "approve requires request", command: approveCmd, wantErr: "accepts 1 arg(s)"},
		{name: "deny accepts request", command: denyCmd, args: []string{"ar-12345678"}},
		{name: "deny requires request", command: denyCmd, wantErr: "accepts 1 arg(s)"},
		{name: "revoke accepts request", command: revokeCmd, args: []string{"ar-12345678"}},
		{name: "revoke requires request", command: revokeCmd, wantErr: "accepts 1 arg(s)"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.command.ValidateArgs(testCase.args)
			if testCase.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestRequiredValue(t *testing.T) {
	value, err := requiredValue("access request ID", "  ar-12345678  ")
	require.NoError(t, err)
	require.Equal(t, "ar-12345678", value)

	_, err = requiredValue("access request ID", " \t ")
	require.ErrorContains(t, err, "access request ID is required")
}

func TestOutputFormat(t *testing.T) {
	for _, testCase := range []struct {
		value   string
		wantErr bool
	}{
		{value: "text"},
		{value: "table"},
		{value: "json"},
		{value: "yaml", wantErr: true},
	} {
		command := &cobra.Command{}
		command.Flags().String(common.OutputFlag, testCase.value, "output")
		output, err := outputFormat(command)
		if testCase.wantErr {
			require.ErrorContains(t, err, "unsupported output format")
			continue
		}
		require.NoError(t, err)
		require.Equal(t, testCase.value, output)
	}
}
