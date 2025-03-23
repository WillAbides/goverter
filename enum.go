package goverter

import (
	"go/types"
	"regexp"
	"sort"
)

type enumConfig struct {
	unknown  string
	enabled  bool
	excludes enumIDPatterns
}

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

type enumIDPattern struct {
	Path *regexp.Regexp
	Name *regexp.Regexp
}

type enumIDPatterns []enumIDPattern

func (ids enumIDPatterns) Matches(path, name string) bool {
	for _, id := range ids {
		if id.Path.MatchString(path) && id.Name.MatchString(name) {
			return true
		}
	}
	return false
}
