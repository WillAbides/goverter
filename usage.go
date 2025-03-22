package goverter

import (
	"sort"
)

type UsageChecker map[string]struct{}

func (u UsageChecker) Used(key string) {
	delete(u, key)
}

func (u UsageChecker) Unused() []string {
	var keys []string
	for key := range u {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
