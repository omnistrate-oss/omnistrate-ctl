package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromPtr(t *testing.T) {
	assert := assert.New(t)

	var strPtr *string
	var intPtr *int

	assert.Equal("test", FromPtr(ToPtr("test")))
	assert.Equal("", FromPtr(strPtr))
	assert.Equal(11, FromPtr(ToPtr(11)))
	assert.Equal(0, FromPtr(intPtr))
}
