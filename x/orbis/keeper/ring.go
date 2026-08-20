package keeper

import "slices"

func canonicalStrings(values []string) []string {
	canonical := slices.Clone(values)
	if !slices.IsSorted(canonical) {
		slices.Sort(canonical)
	}
	return canonical
}
