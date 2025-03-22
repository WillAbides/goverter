package enum

import (
	"go/types"
	"regexp"
)

type Enum struct {
	Type    *types.Named
	Members map[string]any
}

type EnumConfig struct {
	Unknown  string
	Enabled  bool
	Excludes EnumIDPatterns
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

