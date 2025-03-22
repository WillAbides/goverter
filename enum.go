package goverter

import (
	"go/types"
	"regexp"
	"sort"
)

type Enum struct {
	Type    *types.Named
	Members map[string]any
	OK      bool
}

func (e Enum) SortedMembers() []string {
	var m []string
	for member := range e.Members {
		m = append(m, member)
	}
	sort.Strings(m)
	return m
}

type EnumIDPattern struct {
	Path *regexp.Regexp
	Name *regexp.Regexp
}

type EnumIDPatterns []EnumIDPattern

func (ids EnumIDPatterns) Matches(path, name string) bool {
	for _, id := range ids {
		if id.Path.MatchString(path) && id.Name.MatchString(name) {
			return true
		}
	}
	return false
}
