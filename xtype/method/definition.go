package method

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

type Definition struct {
	Parameters
	OriginID string
	Call     *jen.Statement
	ID       string
	Package  string
	Name     string

	Generated  bool
	CustomCall *jen.Statement
}

type Parameters struct {
	TypeParams bool

	Source       *goverter.Type
	MultiSources []*goverter.Type
	Target       *goverter.Type
	Context      map[string]*goverter.Type

	Signature goverter.Signature

	RawArgs []Arg

	ReturnError  bool
	UpdateTarget bool
}

type Arg struct {
	Name string
	Use  ArgUse
	Type *goverter.Type
}

type ArgUse string

const (
	ArgUseSource      ArgUse = "source"
	ArgUseMultiSource ArgUse = "additional-source"
	ArgUseInterface   ArgUse = "interface"
	ArgUseContext     ArgUse = "context"
	ArgUseTarget      ArgUse = "target"
)
