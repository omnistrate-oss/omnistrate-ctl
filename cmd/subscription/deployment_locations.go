package subscription

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

const allowedDeploymentLocationsFlag = "allowed-deployment-locations"

func parseAllowedDeploymentLocations(value string) ([]openapiclientfleet.SubscriptionAllowedDeploymentLocation, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("--%s must be a JSON array, for example '[{\"cloudProvider\":\"aws\",\"regions\":[\"us-east-1\"]}]' or '[]'", allowedDeploymentLocationsFlag)
	}

	var locations []openapiclientfleet.SubscriptionAllowedDeploymentLocation
	decoder := json.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&locations); err != nil {
		return nil, fmt.Errorf("failed to parse --%s as a JSON array: %w", allowedDeploymentLocationsFlag, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil {
		return nil, fmt.Errorf("failed to parse --%s as a JSON array: multiple JSON values provided", allowedDeploymentLocationsFlag)
	} else if err != io.EOF {
		return nil, fmt.Errorf("failed to parse --%s as a JSON array: %w", allowedDeploymentLocationsFlag, err)
	}

	return locations, nil
}
