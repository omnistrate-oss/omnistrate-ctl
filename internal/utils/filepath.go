package utils

import (
	"path/filepath"
	"strings"
)

// GetFirstDifferentSegmentInFilePaths finds the first different segment between targetPath
// and each path in comparisonPaths. If no difference is found, it returns the last segment of targetPath.
func GetFirstDifferentSegmentInFilePaths(targetPath string, comparisonPaths []string) string {
	// Normalize and split the target path
	targetSegments := strings.Split(filepath.Clean(targetPath), string(filepath.Separator))

	if len(comparisonPaths) == 1 {
		return targetSegments[0]
	}

	for _, path := range comparisonPaths {
		if path == targetPath {
			continue // Skip if it's the same path
		}

		// Normalize and split the comparison path
		otherSegments := strings.Split(filepath.Clean(path), string(filepath.Separator))

		// Find the first differing segment
		for i := 0; i < len(targetSegments) && i < len(otherSegments); i++ {
			if targetSegments[i] != otherSegments[i] {
				return targetSegments[i]
			}
		}
	}

	// Return the last segment if all paths are identical up to that point
	return targetSegments[len(targetSegments)-1]
}
