package method

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

type MethodDefinition struct {
	goverter.Parameters
	OriginID string
	Call     *jen.Statement
	ID       string
	Package  string
	Name     string

	Generated  bool
	CustomCall *jen.Statement
}
