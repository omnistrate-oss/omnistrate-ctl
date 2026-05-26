package utils

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestParseDeploymentTarget(t *testing.T) {
	provider, region, ok := ParseDeploymentTarget("Step 2/2: Using cloud provider 'aws' and region 'us-east-1'")

	require.True(t, ok)
	require.Equal(t, "aws", provider)
	require.Equal(t, "us-east-1", region)
}

func TestRenderRegionGlobeIncludesRegionAndMarker(t *testing.T) {
	view := RenderRegionGlobeWithProvider("aws", "us-east-1", 64, true)
	dimView := RenderRegionGlobeWithProvider("aws", "us-east-1", 64, false)

	require.NotContains(t, view, "Global target")
	require.Contains(t, view, "aws")
	require.Contains(t, view, "us-east-1")
	require.Contains(t, view, "╭")
	require.Contains(t, view, "╰")
	require.Contains(t, view, "◉")
	require.Contains(t, dimView, "●")
	require.Contains(t, view, ".")
	require.Contains(t, view, "#")
	require.LessOrEqual(t, maxRenderedLineWidth(view), 64)
}

func TestRegionGlobeSamplesFirehoseBitmap(t *testing.T) {
	require.False(t, isRegionGlobeLand(-160, 85))
	require.True(t, isRegionGlobeLand(-78, 38))
}

func TestRegionGlobeProjectsDifferentRegionsToDifferentPoints(t *testing.T) {
	usX, usY := projectRegionPointFallback(regionPoint("us-east-1"))
	asiaX, asiaY := projectRegionPointFallback(regionPoint("ap-southeast-1"))

	require.NotEqual(t, [2]int{usX, usY}, [2]int{asiaX, asiaY})
	require.Less(t, usX, asiaX)
}

func maxRenderedLineWidth(view string) int {
	maxWidth := 0
	for _, line := range strings.Split(view, "\n") {
		width := lipgloss.Width(line)
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}
