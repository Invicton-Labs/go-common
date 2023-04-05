package collections

// HashMap is an interface that represents a keys-only map. It is useful
// for tracking unique IDs and checking if they exist in a high-performance
// way.
type HashMap[T comparable] interface {
	// Store will store a key in the hash map.
	Store(key T)
	// Delete will delete a key from the hash map if it exists, and returns a bool of whether
	// the key existed and was deleted.
	Delete(key T) bool
	// Has returns a bool of whether the key exists in the hash map.
	Has(key T) bool
	// Length returns the length of the hash map
	Length() int
	// Keys returns a slice of all keys in the hash map.
	Keys() []T
}

type hashMap[T comparable] map[T]struct{}

func NewHashMap[T comparable](initial []T) HashMap[T] {
	initLen := 0
	if initial != nil {
		initLen = len(initial)
	}
	hm := make(hashMap[T], initLen)
	for _, v := range initial {
		hm[v] = struct{}{}
	}
	return hm
}

func NewHashMapPreallocated[T comparable](size int) HashMap[T] {
	return make(hashMap[T], size)
}

func (hm hashMap[T]) Store(key T) {
	hm[key] = struct{}{}
}

func (hm hashMap[T]) Delete(key T) bool {
	_, exists := hm[key]
	if exists {
		delete(hm, key)
	}
	return exists
}

func (hm hashMap[T]) Has(key T) bool {
	_, ok := hm[key]
	return ok
}

func (hm hashMap[T]) Length() int {
	return len(hm)
}

func (hm hashMap[T]) Keys() []T {
	keys := make([]T, 0, len(hm))
	for k := range hm {
		keys = append(keys, k)
	}
	return keys
}
