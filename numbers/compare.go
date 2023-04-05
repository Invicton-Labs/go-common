package numbers

import (
	"math"

	"github.com/Invicton-Labs/go-common/constraints"
)

func Min[T constraints.Ordered](val1 T, vals ...T) T {
	m := val1
	for _, v := range vals {
		if v < m {
			m = v
		}
	}
	return m
}

func Max[T constraints.Ordered](val1 T, vals ...T) T {
	m := val1
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

func IsNaN[T constraints.Float](x T) bool {
	return x != x
}

func IsInf[FT ~float32 | ~float64, ST constraints.Signed](f FT, sign ST) bool {
	switch v := any(f).(type) {
	case float32:
		return sign >= 0 && v > math.MaxFloat32 || sign <= 0 && v < -math.MaxFloat32
	case float64:
		return sign >= 0 && v > math.MaxFloat64 || sign <= 0 && v < -math.MaxFloat64
	default:
		panic("Unexpected type")
	}
}

func Floor[T constraints.Float](x T) int64 {
	r := math.Floor(float64(x))
	if r == 0 {
		return 0
	}
	if math.IsNaN(r) {
		panic("NaN")
	}
	if IsInf(r, 0) {
		panic("Inf")
	}
	return int64(r)
}

func Ceil[T constraints.Float](x T) int64 {
	return -Floor(-x)
}

func Round[T constraints.Float](x T) int64 {
	r := math.Round(float64(x))
	if r == 0 {
		return 0
	}
	if math.IsNaN(r) {
		panic("NaN")
	}
	if IsInf(r, 0) {
		panic("Inf")
	}
	return int64(r)
}
