package collections

func ForEachSlice[Value any](in []Value, f func(value Value)) {
	for _, v := range in {
		f(v)
	}
}

func ForEachMap[Key comparable, Value any](in map[Key]Value, f func(key Key, value Value)) {
	for k, v := range in {
		f(k, v)
	}
}
