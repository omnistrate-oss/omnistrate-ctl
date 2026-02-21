package common

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

// GetInstance resolves an instance ID to its serviceID, environmentID, productTierID, and resourceID
// using the search inventory API.
func GetInstance(ctx context.Context, token, instanceID string) (serviceID, environmentID, productTierID, resourceID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return
	}

	var found bool
	for _, instance := range searchRes.ResourceInstanceResults {
		if instance.Id == instanceID {
			serviceID = instance.ServiceId
			environmentID = instance.ServiceEnvironmentId
			productTierID = instance.ProductTierId
			if instance.ResourceId != nil {
				resourceID = *instance.ResourceId
			}
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("%s not found. Please check the instance ID and try again", instanceID)
		return
	}

	return
}
