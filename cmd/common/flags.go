package common

import (
	"encoding/json"
	"os"
)

const (
	OutputFlag string = "output"

	OutputTypeJson string = "json"
)

func FormatParams(param, paramFile string) (formattedParams map[string]any, err error) {
	// Read parameters from file if provided
	if paramFile != "" {
		var fileContent []byte
		fileContent, err = os.ReadFile(paramFile)
		if err != nil {
			return
		}
		param = string(fileContent)
	}

	// Extract parameters from json format param
	if param != "" {
		err = json.Unmarshal([]byte(param), &formattedParams)
		if err != nil {
			return
		}
	}

	return
}
