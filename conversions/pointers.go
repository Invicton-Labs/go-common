package conversions

func GetPtr[T any](v T) *T {
	return &v
}
