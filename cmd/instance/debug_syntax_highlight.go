package instance

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Syntax highlighting styles
var (
	hclKeywordStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // pink - keywords
	hclStringStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // green - strings
	hclCommentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // gray - comments
	hclNumberStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("178")) // yellow - numbers
	hclBoolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("178")) // yellow - bools
	hclTypeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // blue - types/blocks
	hclFuncStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // blue - functions
	hclAttrStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // white - attributes
	hclOperatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // dim - operators
	hclBraceStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // dim - braces
	hclInterpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // pink - interpolation ${}
	hclVarRefStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // blue - var.xxx references
)

var (
	hclBlockTypes = map[string]bool{
		"resource": true, "data": true, "variable": true, "output": true,
		"locals": true, "module": true, "provider": true, "terraform": true,
		"backend": true, "provisioner": true, "lifecycle": true, "dynamic": true,
		"content": true, "moved": true, "import": true, "check": true,
	}
	hclBuiltinFuncs = map[string]bool{
		"lookup": true, "element": true, "length": true, "split": true,
		"join": true, "format": true, "replace": true, "lower": true,
		"upper": true, "trimspace": true, "tolist": true, "toset": true,
		"tomap": true, "try": true, "can": true, "coalesce": true,
		"concat": true, "contains": true, "distinct": true, "flatten": true,
		"keys": true, "values": true, "merge": true, "zipmap": true,
		"file": true, "fileexists": true, "templatefile": true, "jsonencode": true,
		"jsondecode": true, "yamlencode": true, "yamldecode": true, "base64encode": true,
		"base64decode": true, "cidrsubnet": true, "cidrhost": true, "max": true,
		"min": true, "abs": true, "ceil": true, "floor": true, "log": true,
		"signum": true, "substr": true, "title": true, "chomp": true, "indent": true,
		"regex": true, "regexall": true, "compact": true, "chunklist": true,
		"range": true, "reverse": true, "setintersection": true, "setproduct": true,
		"setsubtract": true, "setunion": true, "sort": true, "sum": true,
		"transpose": true, "matchkeys": true, "one": true, "sensitive": true,
		"nonsensitive": true, "type": true, "tobool": true, "tonumber": true,
		"tostring": true, "startswith": true, "endswith": true, "strrev": true,
		"plantimestamp": true, "uuid": true, "uuidv5": true, "bcrypt": true,
		"md5": true, "sha1": true, "sha256": true, "sha512": true,
	}

	hclNumberRegex = regexp.MustCompile(`^-?\d+\.?\d*$`)
)

// highlightHCLLine applies syntax highlighting to a single HCL line
func highlightHCLLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Full-line comment
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
		return hclCommentStyle.Render(line)
	}

	// Block comment start
	if strings.HasPrefix(trimmed, "/*") {
		return hclCommentStyle.Render(line)
	}

	// Preserve leading whitespace
	leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	return leading + highlightHCLTokens(strings.TrimLeft(line, " \t"))
}

func highlightHCLTokens(line string) string {
	var result strings.Builder
	i := 0
	runes := []rune(line)
	n := len(runes)

	for i < n {
		ch := runes[i]

		// Inline comment
		if ch == '#' || (ch == '/' && i+1 < n && runes[i+1] == '/') {
			result.WriteString(hclCommentStyle.Render(string(runes[i:])))
			return result.String()
		}

		// Block comment
		if ch == '/' && i+1 < n && runes[i+1] == '*' {
			result.WriteString(hclCommentStyle.Render(string(runes[i:])))
			return result.String()
		}

		// Strings (double-quoted, with interpolation highlighting)
		if ch == '"' {
			str, advance := consumeHCLString(runes, i)
			result.WriteString(str)
			i += advance
			continue
		}

		// Braces/brackets
		if ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == '(' || ch == ')' {
			result.WriteString(hclBraceStyle.Render(string(ch)))
			i++
			continue
		}

		// Operators
		if ch == '=' || ch == '!' || ch == '<' || ch == '>' || ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' || ch == '?' || ch == ':' {
			result.WriteString(hclOperatorStyle.Render(string(ch)))
			i++
			continue
		}

		// Whitespace
		if ch == ' ' || ch == '\t' {
			result.WriteRune(ch)
			i++
			continue
		}

		// Commas, dots
		if ch == ',' || ch == '.' {
			result.WriteString(hclOperatorStyle.Render(string(ch)))
			i++
			continue
		}

		// Words (identifiers, keywords, numbers)
		if isWordChar(ch) {
			word, advance := consumeWord(runes, i)

			// Check if it's a function call (followed by open paren)
			nextNonSpace := i + advance
			for nextNonSpace < n && runes[nextNonSpace] == ' ' {
				nextNonSpace++
			}
			isFunc := nextNonSpace < n && runes[nextNonSpace] == '('

			switch {
			case hclBlockTypes[word]:
				result.WriteString(hclTypeStyle.Render(word))
			case word == "true" || word == "false" || word == "null":
				result.WriteString(hclBoolStyle.Render(word))
			case word == "for" || word == "for_each" || word == "if" || word == "in" || word == "each" || word == "count" || word == "depends_on" || word == "source":
				result.WriteString(hclKeywordStyle.Render(word))
			case hclNumberRegex.MatchString(word):
				result.WriteString(hclNumberStyle.Render(word))
			case isFunc && hclBuiltinFuncs[word]:
				result.WriteString(hclFuncStyle.Render(word))
			case isFunc:
				result.WriteString(hclFuncStyle.Render(word))
			case word == "var" || word == "local" || word == "data" || word == "module" || word == "self" || word == "path":
				result.WriteString(hclVarRefStyle.Render(word))
			default:
				result.WriteString(hclAttrStyle.Render(word))
			}
			i += advance
			continue
		}

		// Anything else
		result.WriteRune(ch)
		i++
	}

	return result.String()
}

// consumeHCLString handles a double-quoted string with ${} interpolation highlighting
func consumeHCLString(runes []rune, start int) (string, int) {
	var result strings.Builder
	i := start + 1 // skip opening quote
	n := len(runes)

	result.WriteString(hclStringStyle.Render("\""))

	for i < n {
		ch := runes[i]

		if ch == '\\' && i+1 < n {
			result.WriteString(hclStringStyle.Render(string(runes[i : i+2])))
			i += 2
			continue
		}

		if ch == '"' {
			result.WriteString(hclStringStyle.Render("\""))
			i++
			return result.String(), i - start
		}

		// Interpolation: ${ ... }
		if ch == '$' && i+1 < n && runes[i+1] == '{' {
			result.WriteString(hclInterpStyle.Render("${"))
			i += 2
			depth := 1
			for i < n && depth > 0 {
				if runes[i] == '{' {
					depth++
					result.WriteString(hclInterpStyle.Render("{"))
					i++
				} else if runes[i] == '}' {
					depth--
					if depth == 0 {
						result.WriteString(hclInterpStyle.Render("}"))
					} else {
						result.WriteString(hclInterpStyle.Render("}"))
					}
					i++
				} else if runes[i] == '"' {
					// Nested string inside interpolation
					innerStr, advance := consumeHCLString(runes, i)
					result.WriteString(innerStr)
					i += advance
				} else if isWordChar(runes[i]) {
					word, advance := consumeWord(runes, i)
					switch {
					case word == "var" || word == "local" || word == "data" || word == "module" || word == "each" || word == "self" || word == "count" || word == "path":
						result.WriteString(hclVarRefStyle.Render(word))
					case hclBuiltinFuncs[word]:
						result.WriteString(hclFuncStyle.Render(word))
					case word == "true" || word == "false" || word == "null":
						result.WriteString(hclBoolStyle.Render(word))
					case hclNumberRegex.MatchString(word):
						result.WriteString(hclNumberStyle.Render(word))
					default:
						result.WriteString(hclAttrStyle.Render(word))
					}
					i += advance
				} else {
					if runes[i] == '.' || runes[i] == ',' {
						result.WriteString(hclOperatorStyle.Render(string(runes[i])))
					} else {
						result.WriteRune(runes[i])
					}
					i++
				}
			}
			continue
		}

		result.WriteString(hclStringStyle.Render(string(ch)))
		i++
	}

	// Unterminated string
	return result.String(), i - start
}

func consumeWord(runes []rune, start int) (string, int) {
	i := start
	for i < len(runes) && isWordChar(runes[i]) {
		i++
	}
	return string(runes[start:i]), i - start
}

func isWordChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

// highlightJSONLine applies basic JSON syntax highlighting
func highlightJSONLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}

	leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return leading + highlightJSONTokens(strings.TrimLeft(line, " \t"))
}

func highlightJSONTokens(line string) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0
	n := len(runes)

	for i < n {
		ch := runes[i]

		if ch == '"' {
			// Check if this is a key (followed by : after closing quote)
			str, advance, isKey := consumeJSONString(runes, i)
			if isKey {
				result.WriteString(hclAttrStyle.Render(str))
			} else {
				result.WriteString(hclStringStyle.Render(str))
			}
			i += advance
			continue
		}

		if ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			result.WriteString(hclBraceStyle.Render(string(ch)))
			i++
			continue
		}

		if ch == ':' || ch == ',' {
			result.WriteString(hclOperatorStyle.Render(string(ch)))
			i++
			continue
		}

		if isWordChar(ch) {
			word, advance := consumeWord(runes, i)
			switch word {
			case "true", "false", "null":
				result.WriteString(hclBoolStyle.Render(word))
			default:
				if hclNumberRegex.MatchString(word) {
					result.WriteString(hclNumberStyle.Render(word))
				} else {
					result.WriteString(hclAttrStyle.Render(word))
				}
			}
			i += advance
			continue
		}

		result.WriteRune(ch)
		i++
	}

	return result.String()
}

func consumeJSONString(runes []rune, start int) (string, int, bool) {
	i := start + 1
	n := len(runes)
	for i < n {
		if runes[i] == '\\' && i+1 < n {
			i += 2
			continue
		}
		if runes[i] == '"' {
			i++
			str := string(runes[start:i])
			// Check if followed by : (making it a key)
			j := i
			for j < n && runes[j] == ' ' {
				j++
			}
			isKey := j < n && runes[j] == ':'
			return str, i - start, isKey
		}
		i++
	}
	return string(runes[start:]), n - start, false
}

// syntaxHighlightLine dispatches to the appropriate highlighter based on file extension
func syntaxHighlightLine(line, filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".tf"), strings.HasSuffix(lower, ".tfvars"), strings.HasSuffix(lower, ".hcl"):
		return highlightHCLLine(line)
	case strings.HasSuffix(lower, ".json"):
		return highlightJSONLine(line)
	case strings.HasSuffix(lower, ".sh"):
		return highlightShellLine(line)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(line)
	}
}

// highlightShellLine applies basic shell script highlighting
func highlightShellLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return hclCommentStyle.Render(line)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(line)
}
