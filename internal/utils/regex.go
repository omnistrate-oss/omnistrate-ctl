package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func ReplaceBuildContext(input string, dockerPathsToImageUrls map[string]string) string {
	// Define the pattern to match the build section in the YAML structure.
	// This pattern captures the indentation of the build section as well.
	pattern := `(?m)(^\s*)build:\s*\n\s*context:\s*(?P<context>.+)\n\s*dockerfile:\s*(?P<dockerfile>.+)`

	// Compile the regular expression.
	re := regexp.MustCompile(pattern)

	// Use FindAllStringSubmatch to find all matches in the input text.
	updated := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract the indentation, context, and dockerfile values.
		indentation := ""
		context := ""
		dockerfile := ""

		// Extract the indentation, context, and dockerfile using regex capture groups.
		submatches := re.FindStringSubmatch(match)
		if len(submatches) > 0 {
			indentation = submatches[1]
			context = strings.TrimSpace(submatches[2])
			dockerfile = strings.TrimSpace(submatches[3])
		}

		absContextPath, err := filepath.Abs(context)
		if err != nil {
			return ""
		}
		dockerfilePath := filepath.Join(absContextPath, dockerfile)

		// Construct the replacement string based on the context and dockerfile.
		// Here, we use fmt.Sprintf to generate the images URL and preserve the indentation.
		replacement := fmt.Sprintf(`%simage: "%s"`, indentation, dockerPathsToImageUrls[dockerfilePath])

		return replacement
	})

	// Return the modified input string.
	return updated
}
