package utils

import (
	"testing"
)

func TestGetFirstDifferentSegmentInFilePaths(t *testing.T) {
	tests := []struct {
		targetPath      string
		comparisonPaths []string
		expected        string
	}{
		// Test case 1: Different first segments
		{
			targetPath:      "./frontend/Dockerfile",
			comparisonPaths: []string{"./frontend/Dockerfile", "./backend/Dockerfile"},
			expected:        "frontend",
		},
		// Test case 2: Identical paths
		{
			targetPath:      "./frontend/Dockerfile",
			comparisonPaths: []string{"./frontend/Dockerfile"},
			expected:        "frontend",
		},
		// Test case 3: Different second segments
		{
			targetPath:      "./services/auth/Dockerfile",
			comparisonPaths: []string{"./services/auth/Dockerfile", "./services/api/Dockerfile"},
			expected:        "auth",
		},
		// Test case 4: Paths with multiple common segments, first difference at deeper level
		{
			targetPath:      "./project/frontend/Dockerfile",
			comparisonPaths: []string{"./project/frontend/Dockerfile", "./project/backend/Dockerfile"},
			expected:        "frontend",
		},
		// Test case 5: Different segments in middle of path
		{
			targetPath:      "./project/services/frontend/Dockerfile",
			comparisonPaths: []string{"./project/services/frontend/Dockerfile", "./project/services/backend/Dockerfile"},
			expected:        "frontend",
		},
		// Test case 6: Identical paths with only one segment
		{
			targetPath:      "./Dockerfile",
			comparisonPaths: []string{"./Dockerfile"},
			expected:        "Dockerfile",
		},
	}

	for _, test := range tests {
		t.Run(test.targetPath, func(t *testing.T) {
			result := GetFirstDifferentSegmentInFilePaths(test.targetPath, test.comparisonPaths)
			if result != test.expected {
				t.Errorf("For path %s, expected %s, got %s", test.targetPath, test.expected, result)
			}
		})
	}
}
