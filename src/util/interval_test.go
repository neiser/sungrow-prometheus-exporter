package util

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestIntervals_SortAndMerge(t *testing.T) {
	tests := []struct {
		name     string
		input    Intervals[int]
		expected Intervals[int]
	}{
		{"empty", Intervals[int]{}, Intervals[int]{}},
		{"one element", Intervals[int]{{0, 1}}, Intervals[int]{{0, 1}}},
		{"two disjoint elements", Intervals[int]{{0, 1}, {3, 4}}, Intervals[int]{{0, 1}, {3, 4}}},
		{"three disjoint elements", Intervals[int]{{0, 1}, {3, 4}, {7, 8}}, Intervals[int]{{0, 1}, {3, 4}, {7, 8}}},
		{"two joint elements", Intervals[int]{{0, 1}, {2, 4}}, Intervals[int]{{0, 4}}},
		{"three elements", Intervals[int]{{0, 1}, {6, 7}, {2, 4}}, Intervals[int]{{0, 4}, {6, 7}}},
		{"three subset elements", Intervals[int]{{0, 7}, {1, 3}, {2, 4}}, Intervals[int]{{0, 7}}},
		{"three subset elements and joiner", Intervals[int]{{8, 9}, {0, 7}, {1, 3}, {2, 4}}, Intervals[int]{{0, 9}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.SortAndMerge()
			assert.True(t, reflect.DeepEqual(tt.input, tt.expected), "%v != %v", tt.input, tt.expected)
		})
	}
}
