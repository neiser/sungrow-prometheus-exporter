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

type IntervalSlice[T constraints.Integer, S any] struct {
	Interval[T]
	Slice []S
}

type IntervalSlices[T constraints.Integer, S any] []*IntervalSlice[T, S]

func NewIntervalSlice[T constraints.Integer, S any](interval *Interval[T], slice ...S) *IntervalSlice[T, S] {
	return &IntervalSlice[T, S]{*interval, slice}
}

func (i Interval[T]) String() string {
	return fmt.Sprintf("[%v:%v]", i.Start, i.End)
}

func (i Interval[T]) Contains(v T) bool {
	return v >= i.Start && v <= i.End
}

func (i Interval[T]) Length() T {
	return i.End - i.Start + 1
}

func (intervals *Intervals[T]) SortAndMerge() {
	*intervals = sortAndMerge[T, Interval[T]](*intervals)
}

func (intervalSlices *IntervalSlices[T, S]) SortAndMerge() {
	*intervalSlices = sortAndMerge[T, IntervalSlice[T, S]](*intervalSlices)
}

type interval[T constraints.Integer] interface {
	start() T
	end() T
}

type merger[T constraints.Integer, I interval[T]] interface {
	interval[T]
	consume(other *I)
	append(other *I)
}

func (i Interval[T]) start() T {
	return i.Start
}

func (i Interval[T]) end() T {
	return i.End
}

func (i Interval[T]) consume(*Interval[T]) {
	// do nothing
}

func (i *Interval[T]) append(other *Interval[T]) {
	i.End = other.End
}

func (i IntervalSlice[T, S]) start() T {
	return i.Start
}

func (i IntervalSlice[T, S]) end() T {
	return i.End
}

func (i IntervalSlice[T, S]) consume(*IntervalSlice[T, S]) {
	panic("cannot consume interval slice")
}

func (i *IntervalSlice[T, S]) append(other *IntervalSlice[T, S]) {
	i.End = other.End
	i.Slice = append(i.Slice, other.Slice...)
}

func sortAndMerge[T constraints.Integer, I interval[T], M merger[T, I]](ms []M) (result []M) {
	if len(ms) < 2 {
		return ms
	}
	slices.SortStableFunc(ms, func(a, b M) bool {
		return a.start() < b.start()
	})
	current := ms[0]
	for i := 1; i < len(ms); i++ {
		nextItem := ms[i]
		if current.start() <= nextItem.start() && current.end() >= nextItem.end() {
			current.consume(any(nextItem).(*I))
		} else if current.end()+1 == nextItem.start() {
			current.append(any(nextItem).(*I))
		} else {
			result = append(result, current)
			current = nextItem
		}
	}
	result = append(result, current)
	return
}
