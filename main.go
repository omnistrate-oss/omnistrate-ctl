package main

import "github.com/omnistrate-oss/omnistrate-ctl/cmd"

// IMPORTANT: After modifying any CLI commands, flags, or help text,
// you MUST run `make gen-doc` or `make all` to regenerate documentation.
// This ensures docs in mkdocs/docs/ stay synchronized with CLI behavior.
func main() {
	cmd.Execute()
}
