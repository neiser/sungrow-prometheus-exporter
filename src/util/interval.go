package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type Interval[T constraints.Integer] struct {
	Start, End T
}

type Intervals[T constraints.Integer] []*Interval[T]

type IntervalExt[T constraints.Integer, E any] struct {
	Interval[T]
	Extensions []E
}

type IntervalsExt[T constraints.Integer, E any] []*IntervalExt[T, E]

func (i Interval[T]) String() string {
	return fmt.Sprintf("[%v:%v]", i.Start, i.End)
}

func (i Interval[T]) Contains(v T) bool {
	return v >= i.Start && v <= i.End
}

func (i Interval[T]) Length() T {
	return i.End - i.Start + 1
}

func (intervals Intervals[T]) Sort() {
	slices.SortStableFunc(intervals, func(a, b *Interval[T]) bool {
		return a.Start < b.Start
	})
}

func (intervals *Intervals[T]) SortAndConcat() {
	if len(*intervals) < 2 {
		return
	}
	intervals.Sort()
	var result Intervals[T]
	var current *Interval[T]
	for i := 0; i < len(*intervals)-1; i++ {
		interval := (*intervals)[i]
		nextInterval := (*intervals)[i+1]
		if current == nil {
			current = interval
		}
		if interval.Start <= nextInterval.Start && interval.End >= nextInterval.End {
			continue
		} else if interval.End+1 == nextInterval.Start {
			current.End = nextInterval.End
		} else {
			result = append(result, current)
			current = nextInterval
		}
	}
	if current != nil {
		result = append(result, current)
	}
	*intervals = result
}

func (intervals *IntervalsExt[T, E]) Merge(v T, e E) {
	for _, i := range *intervals {
		if v+1 == i.Start {
			i.Start = v
			i.Extensions = append([]E{e}, i.Extensions...)
			return
		} else if v == i.End+1 {
			i.End = v
			i.Extensions = append(i.Extensions, e)
			return
		}
	}
	*intervals = append(*intervals, &IntervalExt[T, E]{Interval[T]{v, v}, []E{e}})
}
