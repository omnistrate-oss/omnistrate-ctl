package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func ReplaceBuildContext(input string, dockerPathsToImageUrls map[string]string) string {
	// Replace entire build: blocks (including extra keys like cache_from,
	// cache_to) with a single image: line. Block boundaries are determined by
	// indentation: every line indented deeper than the build: key is part of
	// the block.
	contextRe := regexp.MustCompile(`^\s*context:\s*(.+)`)
	dockerfileRe := regexp.MustCompile(`^\s*dockerfile:\s*(.+)`)

	lines := strings.Split(input, "\n")
	var result []string

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimRight(line, " \t")

		// Detect a build: key.
		idx := strings.Index(trimmed, "build:")
		if idx < 0 || strings.TrimSpace(trimmed) != "build:" {
			result = append(result, line)
			i++
			continue
		}

		buildIndent := idx // number of leading spaces before "build:"

		// Collect the child lines of the build: block (indented deeper).
		blockLines := []string{}
		j := i + 1
		for j < len(lines) {
			child := lines[j]
			if strings.TrimSpace(child) == "" {
				// Blank line: peek ahead to see if the next non-blank line
				// is still indented deeper than build: (YAML allows blank
				// lines inside a mapping). If so, skip the blank line and
				// continue consuming the block.
				k := j + 1
				for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
					k++
				}
				if k < len(lines) {
					nextIndent := len(lines[k]) - len(strings.TrimLeft(lines[k], " \t"))
					if nextIndent > buildIndent {
						// Still inside the build block — skip blank lines.
						j = k
						continue
					}
				}
				break
			}
			childIndent := len(child) - len(strings.TrimLeft(child, " \t"))
			if childIndent <= buildIndent {
				break
			}
			blockLines = append(blockLines, child)
			j++
		}

		// Extract context and dockerfile from the block.
		var context, dockerfile string
		for _, bl := range blockLines {
			if m := contextRe.FindStringSubmatch(bl); m != nil {
				context = strings.TrimSpace(m[1])
			}
			if m := dockerfileRe.FindStringSubmatch(bl); m != nil {
				dockerfile = strings.TrimSpace(m[1])
			}
		}

		if context != "" && dockerfile != "" {
			absContextPath, err := filepath.Abs(context)
			if err != nil {
				// Can't resolve path — keep original.
				result = append(result, line)
				i++
				continue
			}
			dockerfilePath := filepath.Join(absContextPath, dockerfile)
			indentation := line[:buildIndent]
			result = append(result, fmt.Sprintf(`%simage: "%s"`, indentation, dockerPathsToImageUrls[dockerfilePath]))
			i = j // skip past the block
		} else {
			// Not a build context block — keep as-is.
			result = append(result, line)
			i++
		}
	}

	return strings.Join(result, "\n")
}
