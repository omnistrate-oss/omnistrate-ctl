package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVPCBaseURL_EscapesAccountConfigID(t *testing.T) {
	// Characters that need URL-path-escaping should be encoded.
	url := vpcBaseURL("ac/123")
	assert.Contains(t, url, "/accountconfig/ac%2F123/cloud-native-networks")

	// Normal IDs should pass through unchanged.
	url = vpcBaseURL("ac-abc-123")
	assert.Contains(t, url, "/accountconfig/ac-abc-123/cloud-native-networks")
}
