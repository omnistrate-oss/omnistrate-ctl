package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("test_key", "test_value")
	value := GetEnv("test_key", "default")
	assert.Equal(t, "test_value", value)
}

func TestGetEnvDefault(t *testing.T) {
	value := GetEnv("inexistent", "default")
	assert.Equal(t, "default", value)
}

func TestGetEnvAsInteger(t *testing.T) {
	os.Setenv("test_key", "1")
	value := GetEnvAsInteger("test_key", "default")
	assert.Equal(t, 1, value)
}

func TestGetEnvAsIntegerDefault(t *testing.T) {
	value := GetEnvAsInteger("inexistent", "1")
	assert.Equal(t, 1, value)
}

func TestGetEnvAsIntegerDefaultNaN(t *testing.T) {
	value := GetEnvAsInteger("inexistent", "default")
	assert.Equal(t, 0, value)
}

func TestGetEnvAsBooleanTrue(t *testing.T) {
	os.Setenv("test_key", "true")
	value := GetEnvAsBoolean("test_key", "false")
	assert.True(t, value)
}

func TestGetEnvAsBooleanFalse(t *testing.T) {
	os.Setenv("test_key", "false")
	value := GetEnvAsBoolean("test_key", "true")
	assert.False(t, value)
}

func TestGetEnvAsBooleanDefault(t *testing.T) {
	os.Setenv("test_key", "")
	value := GetEnvAsBoolean("test_key", "true")
	assert.True(t, value)
}

func TestGetEnvAsBooleanInvalid(t *testing.T) {
	os.Setenv("test_key", "1")
	value := GetEnvAsBoolean("test_key", "true")
	assert.False(t, value)
}

func TestGetEnvAsInt64(t *testing.T) {
	os.Setenv("test_key", "1")
	value := GetEnvAsInt64("test_key", "default")
	assert.Equal(t, int64(1), value)
}

func TestGetEnvAsInt64Default(t *testing.T) {
	value := GetEnvAsInt64("inexistent", "1")
	assert.Equal(t, int64(1), value)
}

func TestGetEnvAsFloat64(t *testing.T) {
	os.Setenv("test_key", "1.1")
	value := GetEnvAsFloat64("test_key", "default")
	assert.Equal(t, float64(1.1), value)
}

func TestGetEnvAsFloat4Default(t *testing.T) {
	value := GetEnvAsFloat64("inexistent", "1.1")
	assert.Equal(t, float64(1.1), value)
}
