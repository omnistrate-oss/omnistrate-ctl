package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/artifactsync"
)

func main() {
	planPath := flag.String("plan", "-", "manual copy plan or managed sync JSON file; - reads stdin")
	vetter := flag.String("vetter", "", "customer vetting executable; receives one JSON object on stdin per staged artifact")
	signer := flag.String("signer", "", "customer signing executable; receives one JSON object on stdin per artifact")
	awsProfile := flag.String("aws-profile", "", "AWS shared-config profile for the destination account")
	flag.Parse()

	reader, closePlan, err := openPlan(*planPath)
	if err != nil {
		fatal(err)
	}
	defer closePlan()
	plan, err := artifactsync.LoadPlan(reader)
	if err == nil {
		err = artifactsync.Run(context.Background(), plan, artifactsync.Options{AWSProfile: *awsProfile, Vetter: *vetter, Signer: *signer})
	}
	if err != nil {
		fatal(err)
	}
}

func openPlan(path string) (io.Reader, func(), error) {
	if path == "" || path == "-" {
		return os.Stdin, func() {}, nil
	}
	file, err := os.Open(path) // #nosec G304 -- the operator explicitly selects the plan file.
	if err != nil {
		return nil, func() {}, err
	}
	return file, func() { _ = file.Close() }, nil
}

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "omnistrate-artifact-sync: %v\n", err)
	os.Exit(1)
}
