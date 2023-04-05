package numbers

import "github.com/Invicton-Labs/go-common/constraints"

// Abs is a generic function for finding the absolute value
func Abs[T constraints.Signed](v T) T {
	if v < 0 {
		return v * -1
	}
	return v
}
