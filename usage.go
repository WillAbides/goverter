package goverter

import (
	"sort"
)

type usageChecker map[string]struct{}

func (u usageChecker) Used(key string) {
	delete(u, key)
}

func (u usageChecker) Unused() []string {
	var keys []string
	for key := range u {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
