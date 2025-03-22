package goverter

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

type ParamType int

const (
	ParamsRequired ParamType = iota
	ParamsOptional
	ParamsNone
)

type ArgUse string

const (
	ArgUseSource      ArgUse = "source"
	ArgUseMultiSource ArgUse = "additional-source"
	ArgUseInterface   ArgUse = "interface"
	ArgUseContext     ArgUse = "context"
	ArgUseTarget      ArgUse = "target"
)

type Arg struct {
	Name string
	Use  ArgUse
	Type *Type
}

type Parameters struct {
	TypeParams bool

	Source       *Type
	MultiSources []*Type
	Target       *Type
	Context      map[string]*Type

	Signature Signature

	RawArgs []Arg

	ReturnError  bool
	UpdateTarget bool
}

type MethodDefinition struct {
	Parameters
	OriginID string
	Call     *jen.Statement
	ID       string
	Package  string
	Name     string

	Generated  bool
	CustomCall *jen.Statement
}

func (def *MethodDefinition) ArgDebug(indent string) string {
	var lines []string
	for _, arg := range def.RawArgs {
		argUse := arg.Use
		if arg.Use == ArgUseMultiSource {
			argUse = ArgUseSource
		} else if arg.Use == ArgUseInterface {
			argUse = ArgUseContext
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", argUse, arg.Type.String))
	}

	if def.Target != nil && !def.UpdateTarget {
		lines = append(lines, fmt.Sprintf("[target] %s", def.Target.String))
	}

	if len(lines) == 0 {
		return ""
	}

	return "\n" + indent + strings.Join(lines, "\n"+indent)
}
