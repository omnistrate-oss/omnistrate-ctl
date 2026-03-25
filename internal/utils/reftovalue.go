package utils

func FromPtr[T any](ref *T) T {
	var emptyValue T
	return FromPtrOrDefault(ref, emptyValue)
}

func FromPtrOrDefault[T any](ref *T, defaultVal T) T {
	if ref != nil {
		return *ref
	}
	return defaultVal
}

// FromInt64Ptr converts int64 pointer to int pointer
func FromInt64Ptr(val *int64) *int {
	if val == nil {
		return nil
	}
	result := int(*val)
	return &result
}
