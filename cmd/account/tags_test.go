package account

import (
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTagsTestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.Flags().String(tagsFlag, "", "")
	return cmd
}

func TestParseAccountTags(t *testing.T) {
	tests := []struct {
		name     string
		value    *string
		want     []openapiclient.CustomTag
		provided bool
		wantErr  string
	}{
		{
			name:  "not provided",
			value: nil,
		},
		{
			name:     "single tag",
			value:    ptr("env=prod"),
			want:     []openapiclient.CustomTag{{Key: "env", Value: "prod"}},
			provided: true,
		},
		{
			name:  "multiple tags sorted by key",
			value: ptr("team=platform, env=prod"),
			want: []openapiclient.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "platform"},
			},
			provided: true,
		},
		{
			name:  "keys with dots and slashes",
			value: ptr("omnistrate.com/cost-center=cc-1234"),
			want: []openapiclient.CustomTag{
				{Key: "omnistrate.com/cost-center", Value: "cc-1234"},
			},
			provided: true,
		},
		{
			name:  "tag value can contain escaped commas",
			value: ptr(`allowed_source_ranges=137.83.246.111/32\,1.2.3.4/32,env=prod`),
			want: []openapiclient.CustomTag{
				{Key: "allowed_source_ranges", Value: "137.83.246.111/32,1.2.3.4/32"},
				{Key: "env", Value: "prod"},
			},
			provided: true,
		},
		{
			name:    "rejects empty value",
			value:   ptr(""),
			wantErr: "at least one tag must be provided in key=value format when using --tags",
		},
		{
			name:    "rejects missing separator",
			value:   ptr("envprod"),
			wantErr: `invalid tag "envprod". Tags must use key=value format`,
		},
		{
			name:    "rejects empty key",
			value:   ptr("=prod"),
			wantErr: "tag key cannot be empty",
		},
		{
			name:    "rejects duplicate keys",
			value:   ptr("env=prod,env=dev"),
			wantErr: "duplicate tag key: env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTagsTestCommand(t)
			if tt.value != nil {
				require.NoError(t, cmd.Flags().Set(tagsFlag, *tt.value))
			}

			tags, provided, err := parseAccountTags(cmd)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.provided, provided)
			assert.Equal(t, tt.want, tags)
		})
	}
}

func TestFormatAccountTags(t *testing.T) {
	assert.Empty(t, formatAccountTags(nil))
	assert.Equal(t, "env=prod,team=platform", formatAccountTags([]openapiclient.CustomTag{
		{Key: "team", Value: "platform"},
		{Key: "env", Value: "prod"},
	}))
}
