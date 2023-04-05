package numbers

import (
	"math"

	"github.com/Invicton-Labs/go-common/constraints"
)

func PowInt[BaseType constraints.Integer, ExpType constraints.Integer](base BaseType, exp ExpType) BaseType {
	if exp < 0 {
		panic("PowInt cannot be used with negative exponents")
	}
	if exp == 0 {
		return 1
	}
	v := base
	var i ExpType
	for i = 1; i < exp; i++ {
		v *= base
	}
	return v
}

func Pow[BaseType constraints.Simple, ExpType constraints.Integer](base BaseType, exp ExpType) float64 {
	return math.Pow(float64(base), float64(exp))
}
