package utility

// Ptr is a helper that returns a pointer to v.
func Ptr[T any](v T) *T {
	return &v
}
