package collections

import (
	"github.com/Invicton-Labs/go-common/constraints"
	"github.com/Invicton-Labs/go-stackerr"
)

// CopyMap creates a copy of the input map
func CopyMap[Key comparable, Value any](in map[Key]Value) map[Key]Value {
	m := make(map[Key]Value, len(in))
	for k, v := range in {
		m[k] = v
	}
	return m
}

// MergeMaps will merge multiple maps together, with values for keys in later maps
// overwriting values with the same keys in previous maps. If no maps are passed
// in, it returns nil. If one map is passed in, it will create a copy of that map.
func MergeMaps[Key comparable, Value any](maps ...map[Key]Value) map[Key]Value {
	if len(maps) == 0 {
		return nil
	}
	out := make(map[Key]Value, len(maps[0]))
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// TransformMap maps an input map to an output map using a transformation function.
func TransformMap[InKey comparable, InValue any, OutKey comparable, OutValue any](in map[InKey]InValue, transformationFunc func(key InKey, value InValue) (transformedKey OutKey, transformedValue OutValue)) map[OutKey]OutValue {
	r := make(map[OutKey]OutValue, len(in))
	for k, v := range in {
		newK, newV := transformationFunc(k, v)
		r[newK] = newV
	}
	return r
}

// TransformMapWithErr maps an input map to an output map using a transformation function that can return an error.
func TransformMapWithErr[InKey comparable, InValue any, OutKey comparable, OutValue any](in map[InKey]InValue, transformationFunc func(key InKey, value InValue) (transformedKey OutKey, transformedValue OutValue, err stackerr.Error)) (map[OutKey]OutValue, stackerr.Error) {
	r := make(map[OutKey]OutValue, len(in))
	for k, v := range in {
		newK, newV, err := transformationFunc(k, v)
		if err != nil {
			return nil, err
		}
		r[newK] = newV
	}
	return r, nil
}

// TransformMapToSlice transforms map into a slice using the given transformation function.
func TransformMapToSlice[MapKeyType comparable, MapValueType any, SliceType any](in map[MapKeyType]MapValueType, transformationFunc func(key MapKeyType, value MapValueType) SliceType) (out []SliceType) {
	out = make([]SliceType, len(in))
	i := 0
	for k, v := range in {
		out[i] = transformationFunc(k, v)
		i++
	}
	return out
}

// MapKeys gets all keys of the input map as a slice.
func MapKeys[Key comparable, Value any](in map[Key]Value) []Key {
	r := make([]Key, len(in))
	idx := 0
	for k := range in {
		r[idx] = k
		idx++
	}
	return r
}

// MapValues gets all values of the input map as a slice.
func MapValues[Key comparable, Value any](in map[Key]Value) []Value {
	r := make([]Value, len(in))
	idx := 0
	for _, v := range in {
		r[idx] = v
		idx++
	}
	return r
}

// MapValuesByAscendingKey gets all values of the input map as a slice, sorted
// in ascending order of the map's keys.
func MapValuesByAscendingKey[Key constraints.Ordered, Value any](in map[Key]Value) []Value {
	keys := MapKeys(in)
	SortSliceAscendingInPlace(keys)
	r := make([]Value, len(in))
	for idx, k := range keys {
		r[idx] = in[k]
		idx++
	}
	return r
}

// MapValuesByDescendingKey gets all values of the input map as a slice, sorted
// in descending order of the map's keys.
func MapValuesByDescendingKey[Key constraints.Ordered, Value any](in map[Key]Value) []Value {
	keys := MapKeys(in)
	SortSliceDescendingInPlace(keys)
	r := make([]Value, len(in))
	for idx, k := range keys {
		r[idx] = in[k]
		idx++
	}
	return r
}

// MapAscending will return a closure (iterator) that will return the next element of the map (in ascending
// order by key) each time it's called. After the last element has been returned, the closure will return
// zero-values and false for 'ok'.
func MapAscending[Key constraints.Ordered, Value any](in map[Key]Value) func() (k Key, v Value, ok bool) {
	keys := MapKeys(in)
	SortSliceAscendingInPlace(keys)
	i := 0
	return func() (k Key, v Value, ok bool) {
		// If we've hit the end, return zero values and false
		if i == len(keys) {
			return k, v, false
		}
		// Increment the counter just before returning
		defer func() { i++ }()
		// Return the key, the value, and true
		return keys[i], in[keys[i]], true
	}
}

// MapDescending will return a closure (iterator) that will return the next element of the map (in descending
// order by key) each time it's called.
func MapDescending[Key constraints.Ordered, Value any](in map[Key]Value) func() (k Key, v Value, ok bool) {
	keys := MapKeys(in)
	SortSliceDescendingInPlace(keys)
	i := 0
	return func() (k Key, v Value, ok bool) {
		// If we've hit the end, return zero values and false
		if i == len(keys) {
			return k, v, false
		}
		// Increment the counter just before returning
		defer func() { i++ }()
		// Return the key, the value, and true
		return keys[i], in[keys[i]], true
	}
}
