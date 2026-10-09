package deploymentcell

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/mitchellh/go-homedir"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

func TestDeploymentCellCommands(t *testing.T) {
	require.Equal(t, "deployment-cell [operation] [flags]", Cmd.Use)
	require.Equal(t, "Manage Deployment Cells", Cmd.Short)
	require.Contains(t, Cmd.Long, "manage Deployment Cells")
	require.Contains(t, Cmd.Commands(), restartCmd)
	require.Equal(t, "restart [deployment-cell-id]", restartCmd.Use)
	require.Equal(t, "Restart a deployment cell deployment", restartCmd.Short)
	require.Contains(t, restartCmd.Long, "does not wait")
	require.Contains(t, restartCmd.Example, "deployment-cell restart hc-12345 --output json")
	require.NotNil(t, restartCmd.RunE)
	require.True(t, restartCmd.SilenceUsage)
	// Restart needs only a host cluster ID; no local flags or request body.
	require.False(t, restartCmd.LocalNonPersistentFlags().HasFlags())
}

func newRestartTestCommand() *cobra.Command {
	command := &cobra.Command{
		Use: restartCmd.Use, Args: restartCmd.Args, RunE: restartCmd.RunE,
		SilenceErrors: true, SilenceUsage: true,
	}
	command.Flags().StringP("output", "o", "table", "Output format")
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	return command
}

func TestRestartRejectsInvalidInputsBeforeAuthentication(t *testing.T) {
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing ID", want: "accepts 1 arg(s), received 0"},
		{name: "too many IDs", args: []string{"hc-one", "hc-two"}, want: "accepts 1 arg(s), received 2"},
		{name: "empty ID", args: []string{""}, want: "deployment cell ID cannot be empty"},
		{name: "whitespace ID", args: []string{" \t\n"}, want: "deployment cell ID cannot be empty"},
		{name: "invalid output", args: []string{"hc-test", "--output", "yaml"}, want: "unsupported output format: yaml"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			command := newRestartTestCommand()
			command.SetArgs(tt.args)
			require.EqualError(t, command.ExecuteContext(t.Context()), tt.want)
		})
	}
}

func TestRestartCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Cleanup(homedir.Reset)
	t.Setenv("OMNISTRATE_API_KEY", "")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
	// Unsigned local fixture, never a real credential.
	token := "mock." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":4102444800}`)) + ".mock"
	require.NoError(t, config.CreateOrUpdateAuthConfig(config.AuthConfig{Token: token}))
	var requests int
	var method, path, auth string
	var body []byte
	var readErr error
	statusCode := http.StatusNoContent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/2022-09-01-00/user" {
			_, _ = w.Write([]byte(`{"id":"mock-user"}`))
			return
		}
		requests++
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		body, readErr = io.ReadAll(r.Body)
		w.WriteHeader(statusCode)
		if statusCode == http.StatusConflict {
			_ = json.NewEncoder(w).Encode(openapiclientfleet.Error{
				Name: "conflict", Message: "restart already in progress",
			})
		}
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)

	for _, output := range []string{"table", "text", "json"} {
		t.Run(output, func(t *testing.T) {
			command := newRestartTestCommand()
			flag := command.Flags().Lookup("output")
			require.Equal(t, "string", flag.Value.Type())
			require.Equal(t, "table", flag.DefValue)
			require.Equal(t, "o", flag.Shorthand)
			command.SetArgs([]string{" hc-test ", "--output", output})
			before := requests
			utils.LastPrintedString = ""
			// Capture stdout to prove JSON has no spinner messages or escape codes.
			reader, writer, err := os.Pipe()
			require.NoError(t, err)
			stdout := os.Stdout
			os.Stdout = writer
			defer func() { os.Stdout = stdout }()
			executeErr := command.ExecuteContext(t.Context())
			os.Stdout = stdout
			require.NoError(t, writer.Close())
			printed, err := io.ReadAll(reader)
			require.NoError(t, reader.Close())
			require.NoError(t, err)
			require.NoError(t, executeErr)
			require.Equal(t, before+1, requests)
			require.Equal(t, http.MethodPost, method)
			require.Equal(t, "/2022-09-01-00/fleet/host-cluster/hc-test/restart-deployment", path)
			require.Equal(t, "Bearer "+token, auth)
			require.NoError(t, readErr)
			require.Empty(t, body)
			require.Contains(t, utils.LastPrintedString, "hc-test")
			require.Contains(t, utils.LastPrintedString, "Deployment restart requested successfully")
			if output == "json" {
				var result model.DeploymentCellRestartResult
				require.NoError(t, json.Unmarshal(printed, &result))
				require.Equal(t, "hc-test", result.ID)
				require.Equal(t, "Deployment restart requested successfully", result.Message)
			}
			require.Equal(t, "table", flag.Value.String())
			require.False(t, flag.Changed)
		})
	}

	t.Run("JSON API error", func(t *testing.T) {
		statusCode = http.StatusConflict
		command := newRestartTestCommand()
		command.SetArgs([]string{"hc-test", "--output", "json"})
		utils.LastPrintedString = ""
		require.EqualError(t, command.ExecuteContext(t.Context()), "conflict\nDetail: restart already in progress")
		require.Empty(t, utils.LastPrintedString)
	})
}
