package util

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func MapSlice[T any, R any](ts []T, mapper func(T) R) (rs []R) {
	for _, t := range ts {
		rs = append(rs, mapper(t))
	}
	return
}

func FindUnequalIndexes[T comparable](a []T, b []T) (unequalIndexes []int) {
	if len(a) != len(b) {
		return nil
	}
	for i := range a {
		if a[i] != b[i] {
			unequalIndexes = append(unequalIndexes, i)
			// continue running to collect all unequal indexes
		}
	}
	return
}

func CompareIgnoring[T constraints.Ordered](indexes []int) func(a, b T) int {
	idx := 0
	return func(a, b T) int {
		defer func() {
			idx++
		}()
		if slices.Contains(indexes, idx) {
			return 0
		}
		return slices.Compare([]T{a}, []T{b})
	}
}
