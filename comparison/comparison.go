package comparison

// PtrEquals will return true if both values are nil,
// or if neither are nil and the values they point
// to are equivalent.
func PtrEquals[T comparable](a *T, b *T) bool {
	if a != nil {
		return b != nil && *a == *b
	}
	return b == nil
}
