package account

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDescribeArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		output  string
		wantErr string
	}{
		{
			name:   "table output is allowed",
			args:   []string{"ac-123"},
			output: "table",
		},
		{
			name:   "text output is allowed",
			args:   []string{"ac-123"},
			output: "text",
		},
		{
			name:   "json output is allowed",
			args:   []string{"ac-123"},
			output: "json",
		},
		{
			name:    "missing account identifier",
			output:  "table",
			wantErr: "account name or ID must be provided",
		},
		{
			name:    "unsupported output",
			args:    []string{"ac-123"},
			output:  "yaml",
			wantErr: "unsupported output format: yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDescribeArguments(tt.args, tt.output)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestPrintDescribeOutput_NebiusTUIFallback(t *testing.T) {
	account := sampleNebiusDescribeAccount()

	output := captureDescribeOutput(t, func() {
		err := printDescribeOutput("table", account)
		require.NoError(t, err)
	})

	for _, expected := range []string{
		"Nebius Account · nebius-account",
		"tenant tenant-e00ezh17k22wmwq5f0",
		"Sections",
		"Overview",
		"Bindings",
		"eu-north1",
		"Account status is READY",
		"1 binding(s)",
		"enter: toggle accordion",
		"c: copy value",
		"Use c to copy the selected value",
	} {
		assert.Contains(t, output, expected)
	}

	assert.NotContains(t, output, "\"nebiusBindings\"")
	assert.Equal(t, strings.TrimRight(output, "\n"), utils.LastPrintedString)
}

func TestPrintDescribeOutput_AWSTUIFallback(t *testing.T) {
	account := sampleAWSDescribeAccount()

	output := captureDescribeOutput(t, func() {
		err := printDescribeOutput("table", account)
		require.NoError(t, err)
	})

	for _, expected := range []string{
		"AWS Account · test-aws",
		"account 123456789012",
		"Overview",
		"Identity",
		"Actions",
		"Bootstrap",
		"Bootstrap (No LB)",
	} {
		assert.Contains(t, output, expected)
	}

	assert.NotContains(t, output, "\"awsAccountID\"")
	assert.Equal(t, strings.TrimRight(output, "\n"), utils.LastPrintedString)
}

func TestWrapAccountDescribeContentPreservesLongURLs(t *testing.T) {
	input := "CloudFormation Template URL: https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/quickcreate?templateURL=https://s3.amazonaws.com/onboarding-cfv1-dev/account-template.yml"

	wrapped := wrapAccountDescribeContent(input, 60)
	lines := strings.Split(wrapped, "\n")
	require.Len(t, lines, 4)
	for _, line := range lines {
		require.LessOrEqual(t, lipgloss.Width(line), 60)
	}
	require.Equal(t, input, strings.ReplaceAll(wrapped, "\n", ""))
}

func TestNebiusBindingAccordionToggleAndSelection(t *testing.T) {
	model := newAccountDescribeModel(sampleNebiusDescribeAccount())

	require.Contains(t, model.viewport.View(), "Account status is READY")
	require.Contains(t, model.View(), "▾ Bindings")
	require.Contains(t, model.View(), "eu-north1")

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Nil(t, cmd)
	updated, ok := updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Equal(t, 1, updated.list.Index())

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	updated, ok = updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Contains(t, updated.View(), "▸ Bindings")
	require.NotContains(t, updated.View(), "• eu-north1")

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	updated, ok = updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Contains(t, updated.View(), "▾ Bindings")
	require.Contains(t, updated.View(), "eu-north1")

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Nil(t, cmd)
	updated, ok = updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Equal(t, 2, updated.list.Index())
	require.Contains(t, updated.viewport.View(), "Project ID: project-e00b497fpr00n5hg8wbh2d")
	require.Contains(t, updated.viewport.View(), "Public key ID matches the configured private key")
	require.Contains(t, updated.viewport.View(), "Key Expires At: 1970-01-01T00:00:00Z")
}

func TestNebiusBindingAccordionTitlePrefersRegionThenProjectID(t *testing.T) {
	require.Equal(t, "eu-north1", nebiusBindingAccordionTitle(openapiclient.NebiusAccountBindingResult{
		Region:    "eu-north1",
		ProjectID: "project-1",
	}, 0))
	require.Equal(t, "project-1", nebiusBindingAccordionTitle(openapiclient.NebiusAccountBindingResult{
		ProjectID: "project-1",
	}, 0))
}

func TestSelectedCopyTextPrefersActionableValue(t *testing.T) {
	item := &accountDescribeItem{
		content:  "Bootstrap\n\nhttps://example.com/template.yml",
		copyText: "https://example.com/template.yml",
	}

	require.Equal(t, "https://example.com/template.yml", selectedCopyText(item))
}

func TestSelectedOpenURLRequiresSingleURL(t *testing.T) {
	require.Equal(t, "https://example.com/template.yml", selectedOpenURL(&accountDescribeItem{
		copyText: "https://example.com/template.yml",
	}))

	require.Empty(t, selectedOpenURL(&accountDescribeItem{
		content: "CloudFormation Template URL: https://example.com/template.yml\nCloudFormation No-LB Template URL: https://example.com/template-no-lb.yml",
	}))
}

func TestSelectedLinkOptionsExtractsMultipleURLs(t *testing.T) {
	options := selectedLinkOptions(&accountDescribeItem{
		title: "Identity",
		content: strings.Join([]string{
			"AWS Identity",
			"",
			"CloudFormation Template URL: https://example.com/template.yml",
			"CloudFormation No-LB Template URL: https://example.com/template-no-lb.yml",
		}, "\n"),
	})

	require.Len(t, options, 2)
	require.Equal(t, "CloudFormation Template URL", options[0].label)
	require.Equal(t, "https://example.com/template.yml", options[0].url)
	require.Equal(t, "CloudFormation No-LB Template URL", options[1].label)
	require.Equal(t, "https://example.com/template-no-lb.yml", options[1].url)
}

func TestOpenSelectedURLShowsPickerForMultiURLSection(t *testing.T) {
	model := newAccountDescribeModel(sampleAWSDescribeAccount())

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Nil(t, cmd)
	updated, ok := updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Equal(t, 1, updated.list.Index())

	updatedAny, cmd = updated.openSelectedURL()
	require.Nil(t, cmd)
	updated, ok = updatedAny.(accountDescribeModel)
	require.True(t, ok)
	require.Len(t, updated.linkOptions, 2)
	require.Contains(t, updated.statusMessage, "Multiple URLs found")
}

func TestPrintDescribeOutput_JSON(t *testing.T) {
	account := &openapiclient.DescribeAccountConfigResult{
		CloudProviderId: "infra-4b57c90820",
		Description:     "Nebius Account tenant-e00ezh17k22wmwq5f0",
		Id:              "ac-9H298ZQqZn",
		Name:            "nebius-account",
		NebiusTenantID:  ptr("tenant-e00ezh17k22wmwq5f0"),
		Status:          "READY",
		StatusMessage:   "account verified",
	}

	output := captureDescribeOutput(t, func() {
		err := printDescribeOutput("json", account)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "\"nebiusTenantID\": \"tenant-e00ezh17k22wmwq5f0\"")
	assert.Contains(t, output, "\"status\": \"READY\"")
	assert.NotContains(t, output, "tab: switch panel")
	assert.Equal(t, strings.TrimRight(output, "\n"), utils.LastPrintedString)
}

func sampleNebiusDescribeAccount() *openapiclient.DescribeAccountConfigResult {
	keyExpiresAt := time.Unix(0, 0).UTC()
	return &openapiclient.DescribeAccountConfigResult{
		CloudProviderId: "infra-4b57c90820",
		Description:     "Nebius Account tenant-e00ezh17k22wmwq5f0",
		Id:              "ac-9H298ZQqZn",
		Name:            "nebius-account",
		NebiusBindings: []openapiclient.NebiusAccountBindingResult{
			{
				DerivedPublicKeyFingerprint: ptr("SHA256:t4Vx7KhZ4T1PMIjgfDE7b1J7PqEsm7M1m+rfS7UhLzI"),
				KeyExpiresAt:                &keyExpiresAt,
				KeyFingerprint:              ptr("b3ab85c336b0fab539e1fd44aaa1c4a6d95cdf691cd64e3bb4b9eaa6b91781c3"),
				KeyState:                    ptr("ACTIVE"),
				ProjectID:                   "project-e00b497fpr00n5hg8wbh2d",
				PublicKeyID:                 "publickey-e00h9scsyy9mbefrjf",
				PublicKeyIDMatches:          ptr(true),
				Region:                      "eu-north1",
				ServiceAccountID:            "serviceaccount-e00vqdp9fskhmmaan8",
				ServiceAccountKeyValidated:  ptr(true),
				Status:                      ptr("READY"),
				StatusMessage:               ptr("binding is ready"),
			},
		},
		NebiusTenantID: ptr("tenant-e00ezh17k22wmwq5f0"),
		Status:         "READY",
		StatusMessage:  "account verified",
	}
}

func sampleAWSDescribeAccount() *openapiclient.DescribeAccountConfigResult {
	return &openapiclient.DescribeAccountConfigResult{
		CloudProviderId:                  "infra-aws",
		Description:                      "AWS account",
		Id:                               "ac-aws",
		Name:                             "test-aws",
		AwsAccountID:                     ptr("123456789012"),
		AwsBootstrapRoleARN:              ptr("arn:aws:iam::123456789012:role/omnistrate-bootstrap-role"),
		AwsCloudFormationTemplateURL:     ptr("https://example.com/template.yml"),
		AwsCloudFormationNoLBTemplateURL: ptr("https://example.com/template-no-lb.yml"),
		Status:                           "READY",
		StatusMessage:                    "account verified",
	}
}

func captureDescribeOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, r)
	require.NoError(t, err)

	return buffer.String()
}
