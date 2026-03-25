package utils

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func GetSupportedFilterKeys[T any](obj T) (supportedFilterKeys []string) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Struct {
		return
	}

	objType := objValue.Type()
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// jsonTag may have options, e.g., "name,omitempty"
			parts := strings.Split(jsonTag, ",")
			jsonTag = parts[0] // Use the first part as the JSON field name
			supportedFilterKeys = append(supportedFilterKeys, jsonTag)
		}
	}
	return
}

func ParseFilters(filters []string, supportedFilterKeys []string) (filterMaps []map[string]string, err error) {
	filterMaps = make([]map[string]string, 0)
	for _, filter := range filters {
		if filter == "[]" {
			continue // Those are empty filters that are reset to default values when clean up args and flags
		}
		filterMap := make(map[string]string)
		filterParts := strings.Split(filter, ",")
		for _, part := range filterParts {
			keyValue := strings.Split(part, ":")
			if len(keyValue) != 2 {
				err = fmt.Errorf("invalid filter format: %s, expected key:value", part)
				return
			}
			if !slices.Contains(supportedFilterKeys, keyValue[0]) {
				err = fmt.Errorf("unsupported filter key: %s", keyValue[0])
				return
			}
			filterMap[keyValue[0]] = keyValue[1]
		}
		filterMaps = append(filterMaps, filterMap)
	}
	return
}

func MatchesFilters[T any](obj T, filterMaps []map[string]string) (matches bool, err error) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Struct {
		return false, fmt.Errorf("obj must be a struct")
	}

	if len(filterMaps) == 0 {
		return true, nil
	}

	// Build a map of JSON tag names to field indices
	objType := objValue.Type()
	jsonTagToFieldIndex := make(map[string]int)
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// jsonTag may have options, e.g., "name,omitempty"
			parts := strings.Split(jsonTag, ",")
			jsonTag = parts[0] // Use the first part as the JSON field name
			jsonTagToFieldIndex[jsonTag] = i
		}
	}

	for _, filterMap := range filterMaps {
		matches = true
		for key, value := range filterMap {
			fieldIndex, found := jsonTagToFieldIndex[key]
			if !found {
				return false, fmt.Errorf("invalid JSON field name: %s", key)
			}

			field := objValue.Field(fieldIndex)
			if !field.IsValid() {
				return false, fmt.Errorf("invalid field index for JSON name: %s", key)
			}

			if !isFieldMatch(field, value) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	return
}

func isFieldMatch(field reflect.Value, value string) bool {
	if !field.IsValid() {
		return false
	}

	var fieldValue string
	switch field.Kind() {
	case reflect.String:
		fieldValue = field.String()
	case reflect.Ptr:
		if field.IsNil() {
			return false
		}
		fieldValue = field.Elem().String()
	default:
		return false
	}

	return strings.EqualFold(fieldValue, value)
}
