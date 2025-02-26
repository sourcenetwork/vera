package property

import "math/rand"

// PickAny returns a random element from ts
// ts must be non empty
func PickAny[T any](ts []T) T {
	i := rand.Int() % len(ts)
	return ts[i]
}
