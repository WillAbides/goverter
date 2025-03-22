package xtype

import (
	"go/constant"
	"go/types"
	"regexp"
	"sort"
)

type Enum struct {
	Type    *types.Named
	Members map[string]any
	OK      bool
}

func (t *Type) Enum(cfg *EnumConfig) *Enum {
	if !t.Named {
		return disabled
	}

	if t.enum == nil {
		t.enum = loadEnum(t.NamedType, cfg)
	}
	return t.enum
}

func loadEnum(t *types.Named, cfg *EnumConfig) *Enum {
	path := t.Obj().Pkg().Path()
	name := t.Obj().Name()

	if !cfg.Enabled || cfg.Excludes.Matches(path, name) {
		return disabled
	}

	e := detectEnum(t)
	return &e
}

func (e Enum) SortedMembers() []string {
	var m []string
	for member := range e.Members {
		m = append(m, member)
	}
	sort.Strings(m)
	return m
}

var disabled = &Enum{OK: false}

func detectEnum(named *types.Named) Enum {
	basic, ok := named.Underlying().(*types.Basic)
	if !ok {
		return Enum{}
	}

	if basic.Info()&(types.IsFloat|types.IsString|types.IsInteger) == 0 {
		return Enum{}
	}

	scope := named.Obj().Pkg().Scope()

	members := map[string]any{}
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok {
			continue
		}

		if types.Identical(named, c.Type()) {
			members[name] = constant.Val(c.Val())
		}
	}

	if len(members) == 0 {
		return Enum{}
	}

	return Enum{
		Type:    named,
		Members: members,
		OK:      true,
	}
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
