package utils

// PtrOf returns a pointer to the given value of any type.
func PtrOf[T any](v T) *T {
	return new(v)
}
