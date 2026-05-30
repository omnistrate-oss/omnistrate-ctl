package utils

import "reflect"

// SnapshotMetadataFromSDK returns snapshotMetadata from a generated SDK model.
// It first checks for a typed SnapshotMetadata field and then falls back to
// AdditionalProperties for SDK versions that do not yet expose the field.
func SnapshotMetadataFromSDK(model interface{}, additionalProperties map[string]interface{}) map[string]interface{} {
	if metadata := snapshotMetadataFromTypedSDKField(model); metadata != nil {
		return metadata
	}
	return SnapshotMetadataFromAdditionalProperties(additionalProperties)
}

// SnapshotMetadataFromAdditionalProperties returns snapshotMetadata preserved by
// the generated SDK for fields that are not yet available as typed model fields.
func SnapshotMetadataFromAdditionalProperties(additionalProperties map[string]interface{}) map[string]interface{} {
	value, ok := additionalProperties["snapshotMetadata"]
	if !ok || value == nil {
		return nil
	}

	return snapshotMetadataMap(value)
}

func snapshotMetadataFromTypedSDKField(model interface{}) map[string]interface{} {
	value := reflect.ValueOf(model)
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	field := value.FieldByName("SnapshotMetadata")
	if !field.IsValid() || field.Kind() != reflect.Map || field.IsNil() {
		return nil
	}

	return snapshotMetadataMap(field.Interface())
}

func snapshotMetadataMap(value interface{}) map[string]interface{} {
	metadata, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	return metadata
}
