package enum

import "regexp"

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
