package servicesorchestration

import (
	"encoding/base64"
	"os"

	"github.com/pkg/errors"
)

// Helper functions

func readDslFile(filePath string) (base64FileContent string, err error) {
	// Read parameters from file if provided
	if filePath == "" {
		err = errors.New("dsl file path is empty")
		return
	}
	var fileContent []byte
	fileContent, err = os.ReadFile(filePath)
	if err != nil {
		return
	}
	// return base64 encoded file content
	base64FileContent = base64.StdEncoding.EncodeToString(fileContent)
	return
}
