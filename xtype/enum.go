package xtype

import (
	"go/constant"
	"go/types"
	"sort"

	"github.com/jmattheis/goverter/enum"
)

type Enum struct {
	enum.Enum
	OK bool
}

func (t *Type) Enum(cfg *enum.EnumConfig) *Enum {
	if !t.Named {
		return disabled
	}

	if t.enum == nil {
		t.enum = loadEnum(t.NamedType, cfg)
	}
	return t.enum
}

func loadEnum(t *types.Named, cfg *enum.EnumConfig) *Enum {
	path := t.Obj().Pkg().Path()
	name := t.Obj().Name()

	if !cfg.Enabled || cfg.Excludes.Matches(path, name) {
		return disabled
	}

	e, ok := detectEnum(t)
	return &Enum{OK: ok, Enum: e}
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

func detectEnum(named *types.Named) (enum.Enum, bool) {
	basic, ok := named.Underlying().(*types.Basic)
	if !ok {
		return enum.Enum{}, false
	}

	if basic.Info()&(types.IsFloat|types.IsString|types.IsInteger) == 0 {
		return enum.Enum{}, false
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
		return enum.Enum{}, false
	}

	return enum.Enum{Type: named, Members: members}, true
}
