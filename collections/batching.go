package collections

import (
	"github.com/Invicton-Labs/go-common/numbers"
)

func Batches[T any](values []T, batchSize int) (batches [][]T) {
	batches = make([][]T, 0, len(values)/batchSize+1)
	for i := 0; i < len(values); i += batchSize {
		batches = append(batches, values[i:numbers.Min(len(values), i+batchSize)])
	}
	return batches
}

func BatchesWithRemainder[T any](values []T, batchSize int) (batches [][]T, remainder []T) {
	batches = Batches(values, batchSize)
	if len(batches) == 0 {
		return [][]T{}, []T{}
	}
	if len(batches[len(batches)-1]) < batchSize {
		remainder = batches[len(batches)-1]
		batches = batches[0 : len(batches)-1]
	} else {
		remainder = []T{}
	}
	return batches, remainder
}

func BatchesIntoChan[T any](values []T, batchSize int, channel chan []T) {
	for i := 0; i < len(values); i += batchSize {
		channel <- values[i:numbers.Min(len(values), i+batchSize)]
	}
}

func BatchesIntoChanWithRemainder[T any](values []T, batchSize int, channel chan []T) (remainder []T) {
	for i := 0; i < len(values); i += batchSize {
		// If the end index is within range
		if i+batchSize <= len(values) {
			// Then add the batch to the channel
			channel <- values[i : i+batchSize]
			// If the end index is the end of the slice
			if i+batchSize == len(values) {
				// Then we're finished, return an empty slice
				return []T{}
			}
		} else {
			// The end index is not within range, so whatever's left is a remainder
			return values[i:]
		}
	}
	// Never reached, but the compiler doesn't know that
	return nil
}
