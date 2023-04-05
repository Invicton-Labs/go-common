package collections

import "github.com/Invicton-Labs/go-common/constraints"

// Range creates a slice of integer values from `start` (inclusive) to
// `end` (exclusive).
func Range[T constraints.Integer](start T, end T) []T {
	r := make([]T, end-start)
	for i := start; i < end; i++ {
		r[i-start] = i
	}
	return r
}
