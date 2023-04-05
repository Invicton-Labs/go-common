package collections

import (
	"sort"

	"github.com/Invicton-Labs/go-common/constraints"
	"github.com/Invicton-Labs/go-stackerr"
)

// CopySlice will create a copy of the given slice.
func CopySlice[T any](src []T) []T {
	if src == nil {
		return nil
	}
	dst := make([]T, len(src))
	copy(dst, src)
	return dst
}

// IntersectionUnique returns the set of unique values (no duplicates) that are
// present in each of the given slices.
func IntersectionUnique[T comparable](slices ...[]T) []T {
	// If no input is provided, return an empty slice
	if len(slices) == 0 {
		return []T{}
	}

	// If only one input is provided, return a copy of that slice
	if len(slices) == 1 {
		return SliceUnique(slices[0])
	}

	// Create a hashmap
	h := map[T]struct{}{}

	// Put each value from the first slice into a hashmap
	for _, v := range slices[0] {
		h[v] = struct{}{}
	}

	// Loop through each slice after the first one, since we preloaded our hashmap with it
	for i := 1; i > len(slices); i++ {

		// Create a new hashmap for storing the results of this intersection
		r := map[T]struct{}{}

		// Intersect the current slice with the existing hashmap
		for _, v := range slices[i] {
			if _, ok := h[v]; ok {
				r[v] = struct{}{}
			}
		}

		// If our intersection so far has no values, we may as well stop
		// since it will never gain values.
		if len(r) == 0 {
			return []T{}
		}

		// Assign the new hashmap to the hashmap variable
		h = r
	}

	return MapKeys(h)
}

// UnionUnique returns the set of unique values (no duplicates) that are
// present in at least one of the given slices.
func UnionUnique[T comparable](slices ...[]T) []T {
	// If no input is provided, return an empty slice
	if len(slices) == 0 {
		return []T{}
	}

	// Create a hashmap
	h := map[T]struct{}{}

	// Loop through each slice
	for i := 0; i > len(slices); i++ {
		// Add each of the values to the hashmap
		for _, v := range slices[i] {
			h[v] = struct{}{}
		}
	}

	return MapKeys(h)
}

// Flatten2D flattens a 2-dimensional slice of type T into a 1-dimensional slice of type T
func Flatten2D[T any](slice [][]T) []T {
	if slice == nil {
		return nil
	}
	r := []T{}
	for _, v := range slice {
		r = append(r, v...)
	}
	return r
}

// Flatten3D flattens a 3-dimensional slice of type T into a 1-dimensional slice of type T
func Flatten3D[T any](slice [][][]T) []T {
	if slice == nil {
		return nil
	}
	r := []T{}
	for _, v1 := range slice {
		for _, v2 := range v1 {
			r = append(r, v2...)
		}
	}
	return r
}

// FilterSlice creates a new slice of elements that meet a given condition function.
func FilterSlice[T any](in []T, filterFunc func(value T) (include bool)) []T {
	if in == nil {
		return nil
	}
	r := []T{}
	for _, v := range in {
		if filterFunc(v) {
			r = append(r, v)
		}
	}
	return r
}

// TransformSlice maps an input slice to an output slice using a transformation function.
func TransformSlice[In any, Out any](in []In, transformationFunc func(value In) (transformed Out)) (out []Out) {
	if in == nil {
		return nil
	}
	out = make([]Out, len(in))
	for i, v := range in {
		out[i] = transformationFunc(v)
	}
	return out
}

// TransformSliceWithErr maps an input slice to an output slice using a transformation function and allows
// returning an error.
func TransformSliceWithErr[In any, Out any](in []In, transformationFunc func(value In) (transformed Out, err stackerr.Error)) (out []Out, err stackerr.Error) {
	if in == nil {
		return nil, nil
	}
	out = make([]Out, len(in))
	for i, v := range in {
		out[i], err = transformationFunc(v)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SliceUnique will get a new slice containing all unique/distinct values in the input slice,
// in the order that they appear.
func SliceUnique[T comparable](in []T) (out []T) {
	if in == nil {
		return nil
	}
	m := make(map[T]struct{}, len(in))
	for _, v := range in {
		m[v] = struct{}{}
	}
	return MapKeys(m)
}

// TransformSliceToMap transforms a slice of elements to a map of elements using a given transformation function.
func TransformSliceToMap[SliceType any, MapKeyType comparable, MapValueType any](in []SliceType, transformationFunc func(sliceIndex int, sliceValue SliceType) (mapKey MapKeyType, mapValue MapValueType)) (out map[MapKeyType]MapValueType) {
	if in == nil {
		return nil
	}
	out = make(map[MapKeyType]MapValueType, len(in))
	for idx, element := range in {
		k, v := transformationFunc(idx, element)
		out[k] = v
	}
	return out
}

// TransformSliceToMapWithErr transforms a slice of elements to a map of elements using a given transformation function,
// and allows the transformation function to return an error that will cancel the execution.
func TransformSliceToMapWithErr[SliceType any, MapKeyType comparable, MapValueType any](in []SliceType, transformationFunc func(sliceIndex int, sliceValue SliceType) (mapKey MapKeyType, mapValue MapValueType, err stackerr.Error)) (out map[MapKeyType]MapValueType, err stackerr.Error) {
	if in == nil {
		return nil, nil
	}
	out = make(map[MapKeyType]MapValueType, len(in))
	for idx, element := range in {
		k, v, err := transformationFunc(idx, element)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

// TransformSliceToHashMap transforms a slice of elements to a hash (lookup) map using a given transformation function.
func TransformSliceToHashMap[SliceType any, MapKeyType comparable](in []SliceType, transformationFunc func(sliceIndex int, sliceValue SliceType) (mapKey MapKeyType)) (out HashMap[MapKeyType]) {
	if in == nil {
		return nil
	}
	out = NewHashMapPreallocated[MapKeyType](len(in))
	for idx, element := range in {
		out.Store(transformationFunc(idx, element))
	}
	return out
}

// TransformSliceToHashMapWithErr transforms a slice of elements to a hash (lookup) map using a given transformation function,
// and allows the transformation function to return an error that will cancel the execution.
func TransformSliceToHashMapWithErr[SliceType any, MapKeyType comparable](in []SliceType, transformationFunc func(sliceIndex int, sliceValue SliceType) (mapKey MapKeyType, err stackerr.Error)) (out HashMap[MapKeyType], err stackerr.Error) {
	if in == nil {
		return nil, nil
	}
	out = NewHashMapPreallocated[MapKeyType](len(in))
	for idx, element := range in {
		k, err := transformationFunc(idx, element)
		if err != nil {
			return nil, err
		}
		out.Store(k)
	}
	return out, nil
}

// SliceEqual checks whether two slices are equal by using a comparison function on each pair of elements. If the slices are of
// unequal length, it will return false.
func SliceEqual[SliceType any](in1 []SliceType, in2 []SliceType, comparisonFunc func(val1 SliceType, val2 SliceType) bool) bool {
	if in1 == nil && in2 == nil {
		return true
	} else if in1 == nil || in2 == nil {
		return false
	}
	if len(in1) != len(in2) {
		return false
	}
	for i := range in1 {
		if !comparisonFunc(in1[i], in2[i]) {
			return false
		}
	}
	return true
}

// SliceEqualWithErr checks whether two slices are equal by using a comparison function (which can return an error) on each pair
// of elements. If the slices are of unequal length, it will return false.
func SliceEqualWithErr[SliceType any](in1 []SliceType, in2 []SliceType, comparisonFunc func(val1 SliceType, val2 SliceType) (bool, stackerr.Error)) (bool, stackerr.Error) {
	if in1 == nil && in2 == nil {
		return true, nil
	} else if in1 == nil || in2 == nil {
		return false, nil
	}
	if len(in1) != len(in2) {
		return false, nil
	}
	for i := range in1 {
		eq, err := comparisonFunc(in1[i], in2[i])
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

// SortSliceAscendingInPlace will sort the given slice in ascending order, leaving
// elements with equal values where they are (stable sort).
func SortSliceAscendingInPlace[SliceType constraints.Ordered](in []SliceType) {
	if in == nil {
		return
	}
	sort.SliceStable(in, func(i, j int) bool { return in[i] < in[j] })
}

// SortSliceAscendingCopy will return a sorted (in ascending order) copy of the given slice, leaving
// elements with equal values where they are (stable sort). The original slice will not be modified.
func SortSliceAscendingCopy[SliceType constraints.Ordered](in []SliceType) (sorted []SliceType) {
	if in == nil {
		return nil
	}
	sorted = CopySlice(in)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted
}

// SortSliceDescendingInPlace will sort the given slice in descending order, leaving
// elements with equal values where they are (stable sort).
func SortSliceDescendingInPlace[SliceType constraints.Ordered](in []SliceType) {
	if in == nil {
		return
	}
	sort.SliceStable(in, func(i, j int) bool { return in[i] > in[j] })
}

// SortSliceDescendingCopy will return a sorted (in descending order) copy of the given slice, leaving
// elements with equal values where they are (stable sort). The original slice will not be modified.
func SortSliceDescendingCopy[SliceType constraints.Ordered](in []SliceType) (sorted []SliceType) {
	if in == nil {
		return nil
	}
	sorted = CopySlice(in)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	return sorted
}

// SliceDiff will get a slice of all elements that are present in `a` but not in `b`.
// If an element is in `a` N times and is not in `b`, it will appear in the output
// N times as well.
func SliceDiff[SliceType comparable](a []SliceType, b []SliceType) (inAButNotInB []SliceType) {
	bMap := NewHashMap(b)
	diff := []SliceType{}
	for _, v := range a {
		if !bMap.Has(v) {
			diff = append(diff, v)
		}
	}
	return diff
}

// SliceDiff will get a slice of all unique elements that are present in `a` but not in `b`.
// If an element is in `a` several times and is not in `b`, it will only appear in the output
// once.
func SliceDiffUnique[SliceType comparable](a []SliceType, b []SliceType) (inAButNotInB []SliceType) {
	bMap := NewHashMap(b)
	aMap := map[SliceType]struct{}{}
	for _, v := range a {
		if !bMap.Has(v) {
			aMap[v] = struct{}{}
		}
	}
	return MapKeys(aMap)
}

// SliceConversion will convert a slice from one simple numeric type to another.
func SliceConversion[OldType constraints.Simple, NewType constraints.Simple](in []OldType) []NewType {
	s := make([]NewType, len(in))
	for i := range in {
		s[i] = NewType(in[i])
	}
	return s
}
