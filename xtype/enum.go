package xtype

import (
	"go/constant"
	"go/types"

	"github.com/jmattheis/goverter"
)

func (t *Type) Enum(cfg *EnumConfig) *goverter.Enum {
	if !t.Named {
		return &goverter.Enum{}
	}

	if t.enum == nil {
		t.enum = loadEnum(t.NamedType, cfg)
	}
	return t.enum
}

func loadEnum(t *types.Named, cfg *EnumConfig) *goverter.Enum {
	path := t.Obj().Pkg().Path()
	name := t.Obj().Name()

	if !cfg.Enabled || cfg.Excludes.Matches(path, name) {
		return &goverter.Enum{}
	}

	e := detectEnum(t)
	return &e
}

func detectEnum(named *types.Named) goverter.Enum {
	basic, ok := named.Underlying().(*types.Basic)
	if !ok {
		return goverter.Enum{}
	}

	if basic.Info()&(types.IsFloat|types.IsString|types.IsInteger) == 0 {
		return goverter.Enum{}
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
		return goverter.Enum{}
	}

	return goverter.Enum{
		Type:    named,
		Members: members,
		OK:      true,
	}
}

type EnumConfig struct {
	Unknown  string
	Enabled  bool
	Excludes goverter.EnumIDPatterns
}
