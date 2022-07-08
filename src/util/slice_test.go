package util

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"testing"
)

func TestCompareIgnoring(t *testing.T) {

}

func TestCompareIgnoring1(t *testing.T) {
	type args struct {
		a, b, ignored []int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"non-ignored", args{[]int{1, 2}, []int{1, 2}, []int{}}, 0},
		{"last ignored", args{[]int{1, 2}, []int{1, 3}, []int{1}}, 0},
		{"unequal non-ignored", args{[]int{1, 2}, []int{1, 3}, []int{}}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, slices.CompareFunc(tt.args.a, tt.args.b, CompareIgnoring[int](tt.args.ignored)), "%v == %v ignoring %v", tt.args.a, tt.args.b, tt.args.ignored)
		})
	}
}
