package main

import (
	"log"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	err := doc.GenMarkdownTree(cmd.RootCmd, "./mkdocs/docs/")
	if err != nil {
		log.Fatal(err)
	}
}
