package utils

// Utility methods to convert value to references in a single line
//  converts a value to a reference

func ToPtr[T any](value T) *T {
	return &value
}
