package collections

import "strings"

func SliceContains[T comparable](slice []T, value T) bool {
	_, found := SliceFind(slice, value)
	return found
}

func SliceContainsCaseInsensitive(slice []string, value string) bool {
	_, found := SliceFindCaseInsensitive(slice, value)
	return found
}

func SliceFind[T comparable](slice []T, value T) (int, bool) {
	for i, v := range slice {
		if v == value {
			return i, true
		}
	}
	return -1, false
}

func SliceFindCaseInsensitive(slice []string, value string) (int, bool) {
	for i, v := range slice {
		if strings.EqualFold(v, value) {
			return i, true
		}
	}
	return -1, false
}
