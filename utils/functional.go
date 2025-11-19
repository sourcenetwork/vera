package utils

import "fmt"

// MapSlice produces a new slice from a slice of elements and a mapping function
func MapSlice[T any, U any](ts []T, mapper func(T) U) []U {
	us := make([]U, 0, len(ts))

	for _, t := range ts {
		u := mapper(t)
		us = append(us, u)
	}

	return us
}

// MapFailableSlice maps a slice using a mapping function which may fail.
// Returns upon all elements are mapped or terminates upon the first mapping error.
func MapFailableSlice[T any, U any](ts []T, mapper func(T) (U, error)) ([]U, error) {
	us := make([]U, 0, len(ts))

	for i, t := range ts {
		u, err := mapper(t)
		if err != nil {
			return nil, fmt.Errorf("slice elem %v: %w", i, err)
		}
		us = append(us, u)
	}

	return us, nil
}

// MapNullableSlice produces a new slice of elements from a slice of pointers, excluding 'nil' elements.
func MapNullableSlice[T any, U any](ts []*T, mapper func(*T) U) []U {
	us := make([]U, 0, len(ts))

	for _, t := range ts {
		if t != nil {
			u := mapper(t)
			us = append(us, u)
		}
	}

	return us
}
