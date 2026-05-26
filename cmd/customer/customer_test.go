package customer

import (
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCustomerCommands(t *testing.T) {
	require.Equal(t, "customer [operation] [flags]", Cmd.Use)
	require.Contains(t, Cmd.Short, "customer portal users")

	expectedCommands := []string{"create", "list", "describe", "update", "verify", "suspend", "unsuspend", "delete"}
	actualCommands := make([]string, 0, len(Cmd.Commands()))
	for _, command := range Cmd.Commands() {
		actualCommands = append(actualCommands, command.Name())
	}

	require.ElementsMatch(t, expectedCommands, actualCommands)
}

func TestCustomerCommandFlags(t *testing.T) {
	require.NotNil(t, createCmd.Flag("email"))
	require.NotNil(t, createCmd.Flag("name"))
	require.NotNil(t, createCmd.Flag("password"))
	require.NotNil(t, createCmd.Flag("password-stdin"))
	require.NotNil(t, createCmd.Flag("legal-company-name"))
	require.NotNil(t, createCmd.Flag("company-url"))
	require.NotNil(t, createCmd.Flag("auto-verify"))
	require.NotNil(t, createCmd.Flag("attribute"))

	require.NotNil(t, listCmd.Flag("next-page-token"))
	require.NotNil(t, listCmd.Flag("page-size"))
	require.NotNil(t, listCmd.Flag("exclude-stats"))

	require.NotNil(t, describeCmd.Flag("user-id"))
	require.NotNil(t, updateCmd.Flag("user-id"))
	require.NotNil(t, updateCmd.Flag("attribute"))
	require.NotNil(t, verifyCmd.Flag("user-id"))
	require.NotNil(t, suspendCmd.Flag("user-id"))
	require.NotNil(t, unsuspendCmd.Flag("user-id"))
	require.NotNil(t, deleteCmd.Flag("user-id"))
}

func TestCustomerCommandsDoNotAskForService(t *testing.T) {
	for _, cmd := range []*cobra.Command{createCmd, listCmd, describeCmd, updateCmd, verifyCmd, suspendCmd, unsuspendCmd, deleteCmd} {
		require.Nil(t, cmd.Flag("service"), cmd.Name())
		require.NotContains(t, cmd.Example, "--service", cmd.Name())
		require.NotContains(t, cmd.Long, "service", cmd.Name())
	}
}

func TestCreateExamplesMentionPasswordStdinAndAutoVerify(t *testing.T) {
	require.Contains(t, createExample, "--password-stdin")
	require.Contains(t, createExample, "--auto-verify")
}

func TestCustomerCreatePassword(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		passwordStdin bool
		stdin         string
		stdinTerminal bool
		want          string
		wantErr       string
	}{
		{
			name:     "password flag",
			password: "  secret  ",
			want:     "secret",
		},
		{
			name:          "stdin password",
			passwordStdin: true,
			stdin:         "  secret\n",
			want:          "secret",
		},
		{
			name:          "mutually exclusive",
			password:      "secret",
			passwordStdin: true,
			stdin:         "other",
			wantErr:       "mutually exclusive",
		},
		{
			name:          "empty stdin",
			passwordStdin: true,
			stdin:         "\n",
			wantErr:       "non-empty password",
		},
		{
			name:          "terminal stdin",
			passwordStdin: true,
			stdinTerminal: true,
			wantErr:       "requires piped or redirected input",
		},
		{
			name:    "missing",
			wantErr: "non-empty password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCustomerCreatePasswordTestCommand(t)
			if tt.password != "" {
				require.NoError(t, cmd.Flags().Set("password", tt.password))
			}
			if tt.passwordStdin {
				require.NoError(t, cmd.Flags().Set("password-stdin", "true"))
			}

			got, err := customerCreatePassword(cmd, strings.NewReader(tt.stdin), tt.stdinTerminal)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func newCustomerCreatePasswordTestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.Flags().String("password", "", "")
	cmd.Flags().Bool("password-stdin", false, "")
	return cmd
}

func TestParseAttributes(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    map[string]string
		wantErr string
	}{
		{
			name:   "empty",
			values: []string{},
			want:   map[string]string{},
		},
		{
			name:   "repeated and comma separated",
			values: []string{"tier=enterprise", "region=us-west-2, owner = platform "},
			want: map[string]string{
				"tier":   "enterprise",
				"region": "us-west-2",
				"owner":  "platform",
			},
		},
		{
			name:    "invalid",
			values:  []string{"missing-equals"},
			wantErr: "Attributes must use key=value format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAttributes(tt.values)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFormatUser(t *testing.T) {
	enabled := true
	subscriptionCount := int64(2)
	instanceCount := int64(3)

	got := formatUser(fleet.AccessSideUser{
		UserId:            stringPointer("user-1"),
		UserName:          stringPointer("Jane Doe"),
		Email:             stringPointer("jane@example.com"),
		Status:            stringPointer("ACTIVE"),
		Enabled:           &enabled,
		OrgId:             stringPointer("org-1"),
		OrgName:           stringPointer("Example"),
		SubscriptionCount: &subscriptionCount,
		InstanceCount:     &instanceCount,
		CreatedAt:         stringPointer("2026-01-01T00:00:00Z"),
	})

	require.Equal(t, "user-1", got.UserID)
	require.Equal(t, "Jane Doe", got.UserName)
	require.Equal(t, "jane@example.com", got.Email)
	require.Equal(t, "ACTIVE", got.Status)
	require.Equal(t, "true", got.Enabled)
	require.Equal(t, "2", got.SubscriptionCount)
	require.Equal(t, "3", got.InstanceCount)
}

func stringPointer(value string) *string {
	return &value
}
