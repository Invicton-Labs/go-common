package zero

// Returns the zero value of a given type.
func ZeroValue[T any]() T {
	var t T
	return t
}

// Returns a pointer to the zero value of a given type
func ZeroValuePtr[T any]() *T {
	var t T
	return &t
}
