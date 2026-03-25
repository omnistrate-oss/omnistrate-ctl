package config

import (
	"os"
	"strconv"
	"strings"
)

func GetEnv(env, defaultValue string) string {
	environment := os.Getenv(env)
	if environment == "" {
		return defaultValue
	}
	return environment
}

func GetEnvAsInteger(envName, defaultValue string) int {
	num, err := strconv.Atoi(GetEnv(envName, defaultValue))
	if err != nil {
		return 0
	}
	return num
}

func GetEnvAsInt64(envName, defaultValue string) int64 {
	num, err := strconv.ParseInt(GetEnv(envName, defaultValue), 10, 64)
	if err != nil {
		return 0
	}
	return num
}

func GetEnvAsFloat64(envName, defaultValue string) float64 {
	num, err := strconv.ParseFloat(GetEnv(envName, defaultValue), 64)
	if err != nil {
		return 0
	}
	return num
}

func GetEnvAsBoolean(envName, defaultValue string) bool {
	return strings.ToLower(GetEnv(envName, defaultValue)) == "true"
}
