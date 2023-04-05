package collections

import (
	"github.com/Invicton-Labs/go-common/constraints"
	"github.com/Invicton-Labs/go-common/zero"
)

// Repeat creates a slice that repeats the given value a certain
// number of times.
func Repeat[T any, C constraints.Integer](value T, count C) []T {
	v := make([]T, count)
	for i := C(0); i < count; i++ {
		v[i] = value
	}
	return v
}

// RepeatZero creates a slice of a certain number of copies of the
// zero value of the given type. Only the first type parameter (T)
// must be provided; the second (C) will be inferred from the input
// argument.
func RepeatZero[T any, C constraints.Integer](count C) []T {
	v := make([]T, count)
	for i := C(0); i < count; i++ {
		v[i] = zero.ZeroValue[T]()
	}
	return v
}

func RepeatDynamic[T any, C constraints.Integer](creationFunc func(index C) T, count C) []T {
	v := make([]T, count)
	for i := C(0); i < count; i++ {
		v[i] = creationFunc(i)
	}
	return v
}
