package keeper

import "slices"

func canonicalNodeKeys(nodeKeys []string) []string {
	canonical := slices.Clone(nodeKeys)
	if !slices.IsSorted(canonical) {
		slices.Sort(canonical)
	}
	return canonical
}
