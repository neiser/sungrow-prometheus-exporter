package util

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestIntervals_SortAndConcat(t *testing.T) {
	tests := []struct {
		name     string
		input    Intervals[int]
		expected Intervals[int]
	}{
		{"empty", Intervals[int]{}, Intervals[int]{}},
		{"one element", Intervals[int]{{0, 1}}, Intervals[int]{{0, 1}}},
		{"two disjoint elements", Intervals[int]{{0, 1}, {3, 4}}, Intervals[int]{{0, 1}, {3, 4}}},
		{"two joint elements", Intervals[int]{{0, 1}, {2, 4}}, Intervals[int]{{0, 4}}},
		{"three elements", Intervals[int]{{0, 1}, {6, 7}, {2, 4}}, Intervals[int]{{0, 4}, {6, 7}}},
		{"three subset elements", Intervals[int]{{0, 7}, {0, 7}, {2, 4}}, Intervals[int]{{0, 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.SortAndConcat()
			assert.True(t, reflect.DeepEqual(tt.input, tt.expected), "%v != %v", tt.input, tt.expected)
		})
	}
}
