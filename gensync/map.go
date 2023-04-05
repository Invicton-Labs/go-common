package gensync

import "sync"

// This is a generically typed version of the built-in sync.Map
type Map[K comparable, V any] struct {
	m sync.Map
}

// NewMap creates a new map with initial values
func NewMap[K comparable, V any](initial map[K]V) *Map[K, V] {
	m := Map[K, V]{}
	for k, v := range initial {
		m.Store(k, v)
	}
	return &m
}

// Delete deletes the value for a key.
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		var zeroValue V
		return zeroValue, ok
	}
	return v.(V), ok
}

// LoadAndDelete deletes the value for a key, returning the previous value if any. The loaded result reports whether the key was present.
func (m *Map[K, V]) LoadAndDelete(key K) (value V, ok bool) {
	v, ok := m.m.LoadAndDelete(key)
	if !ok {
		var zeroValue V
		return zeroValue, ok
	}
	return v.(V), ok
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	if !loaded {
		return v.(V), loaded
	}
	return v.(V), loaded
}

/*
Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.

Range does not necessarily correspond to any consistent snapshot of the Map's contents: no key will be visited more than once, but if the value for any key is stored or deleted concurrently (including by f), Range may reflect any mapping for that key from any point during the Range call. Range does not block other methods on the receiver; even f itself may call any method on m.

Range may be O(N) with the number of elements in the map even if f returns false after a constant number of calls.
*/
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	fAny := func(key, value any) bool {
		return f(key.(K), value.(V))
	}
	m.m.Range(fAny)
}

// Length will get the number of elements in the map. It is subject to the same conditions/restrictions as Range.
func (m *Map[K, V]) Length() (length int) {
	m.Range(func(_ K, _ V) bool {
		length++
		return true
	})
	return length
}

// Keys will get all keys in the map. It is subject to the same conditions/restrictions as Range.
func (m *Map[K, V]) Keys() []K {
	keys := []K{}
	m.Range(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

// Values will get all values in the map. It is subject to the same conditions/restrictions as Range.
func (m *Map[K, V]) Values() []V {
	values := []V{}
	m.Range(func(_ K, value V) bool {
		values = append(values, value)
		return true
	})
	return values
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Has checks if the map contains the given key
func (m *Map[K, V]) Has(key K) bool {
	_, ok := m.m.Load(key)
	return ok
}

// ToMap returns a standard map with all of the values in this sync map.
func (m *Map[K, V]) ToMap() map[K]V {
	nm := map[K]V{}
	m.Range(func(key K, value V) bool {
		nm[key] = value
		return true
	})
	return nm
}
