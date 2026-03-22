package utils

import (
	"sort"
)

var _ sort.Interface = (*Sortable[any])(nil)

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64 | ~string | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Sortable wraps a slice of values with a type that implements sort.Interface.
type Sortable[T any] struct {
	ts         []T
	comparator Comparator[T]
}

// Extractor extracts an Ordered value from a type T
type Extractor[T any, K Ordered] func(T) K

// Comparator is a function which compares whether left is less than right
// returns true if it is else false
type Comparator[T any] func(left, right T) bool

// FromExtractor builds a Sortable from a slice of values and an extractor function.
// The extractor function extracts an Ordered value from the slice elements which is used as the comparassion value.
//
// eg. suppose T represents a Person, the extractor function:
// func (p *Person) string { return p.Name }
// would be used to sort all Persons in the slice by their Name.
func FromExtractor[T any, K Ordered](vals []T, extractor Extractor[T, K]) Sortable[T] {
	comparator := func(left, right T) bool {
		return extractor(left) < extractor(right)
	}

	return Sortable[T]{
		ts:         vals,
		comparator: comparator,
	}
}

// FromComparator creates a Sortable from a comparator function which takes two instances of T and returns
// whether left is less than right
func FromComparator[T any](vals []T, comparator Comparator[T]) Sortable[T] {
	return Sortable[T]{
		ts:         vals,
		comparator: comparator,
	}
}

func (s *Sortable[T]) Len() int {
	return len(s.ts)
}

func (s *Sortable[T]) Swap(i, j int) {
	s.ts[i], s.ts[j] = s.ts[j], s.ts[i]
}

func (s *Sortable[T]) Less(i, j int) bool {
	ti, tj := s.ts[i], s.ts[j]
	return s.comparator(ti, tj)
}

// SortInPlace sorts the original slice supplied when the Sortable was initialized
func (s Sortable[T]) SortInPlace() {
	sort.Stable(&s)
}

// Sort returns a sorted slice of the elements given originally
func (s Sortable[T]) Sort() []T {
	vals := make([]T, len(s.ts))
	copy(vals, s.ts)
	sortable := Sortable[T]{
		ts:         vals,
		comparator: s.comparator,
	}
	sortable.SortInPlace()
	return vals
}

// SortSlice performs an inplace sort of a Slice of Ordered elements
func SortSlice[T Ordered](elems []T) {
	sortable := Sortable[T]{
		ts: elems,
		comparator: func(left T, right T) bool { return left < right },
	}
	sortable.SortInPlace()
}
