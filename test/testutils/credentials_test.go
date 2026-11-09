package testutils

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTestDBPasswordIsStableAndStrong(t *testing.T) {
	first := TestDBPassword()
	second := TestDBPassword()

	require.Equal(t, first, second, "the password must stay the same within one process")
	require.Len(t, first, testDBPasswordLength)
	require.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]+$`), first)
}

func TestRandomTestPasswordIsStrong(t *testing.T) {
	password := RandomTestPassword()

	require.Len(t, password, testDBPasswordLength)
	require.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]+$`), password)
}

func TestRenderTestCredentialsReplacesEveryPlaceholder(t *testing.T) {
	spec := "env:\n  - MYSQL_PASSWORD=" + TestDBPasswordPlaceholder + "\nauth:\n  password: " + TestDBPasswordPlaceholder + "\n"

	rendered := RenderTestCredentials(spec)

	require.NotContains(t, rendered, TestDBPasswordPlaceholder)
	require.Equal(t, 2, strings.Count(rendered, TestDBPassword()))
}

func TestRenderTestCredentialsLeavesSpecsWithoutPlaceholderUnchanged(t *testing.T) {
	spec := "services:\n  web:\n    image: nginx\n"

	require.Equal(t, spec, RenderTestCredentials(spec))
}
