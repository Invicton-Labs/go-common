package dateutils

import (
	"math"
	"time"

	"github.com/Invicton-Labs/go-common/constraints"
	"github.com/Invicton-Labs/go-common/numbers"
)

// TimeFromUnix will parse a Unix timestamp that can be in seconds, milliseconds, microseconds, or nanoseconds.
func TimeFromUnix[T constraints.Integer](unix T) time.Time {
	u := int64(unix)
	magnitude := numbers.Abs(u)
	switch {
	case magnitude <= math.MaxInt32:
		return time.Unix(u, 0)
	case magnitude <= 1e3*math.MaxInt32:
		// It's in milliseconds
		return time.UnixMilli(u)
	case magnitude <= 10e6*math.MaxInt32:
		// It's in microseconds
		return time.UnixMicro(u)
	default:
		// It must be in nanoseconds
		// Use the nanoseconds divided by 1e9 as the seconds,
		// and the remaining nanoseconds.
		return time.Unix(u/1e9, u%1e9)
	}
}
