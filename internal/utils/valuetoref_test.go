package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtr(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	assert.Equal("test", *ToPtr("test"))
	assert.Equal(2, *ToPtr(2))
	assert.Equal(int32(2), *ToPtr(int32(2)))
	assert.Equal(int64(2), *ToPtr(int64(2)))
	assert.Equal(true, *ToPtr(true))
	assert.Equal(float64(2), *ToPtr(float64(2)))
	assert.Equal(float32(2), *ToPtr(float32(2)))
	assert.Equal(uint(2), *ToPtr(uint(2)))
	assert.Equal(uint32(2), *ToPtr(uint32(2)))
	assert.Equal(uint64(2), *ToPtr(uint64(2)))
	assert.Equal(true, *ToPtr(true))
	assert.Equal(float32(0.1), *ToPtr(float32(0.1)))
	assert.Equal(0.1, *ToPtr(0.1))

	type testStruct struct {
		Name string `json:"Name,omitempty"`
	}

	assert.Equal(testStruct{Name: "test"}, *ToPtr(testStruct{Name: "test"}))

	type enumString string

	const (
		EnumString1 enumString = "EnumString1"
		EnumString2 enumString = "EnumString2"
	)

	assert.Equal(EnumString1, *ToPtr(EnumString1))
	assert.Equal(EnumString2, *ToPtr(EnumString2))
}
