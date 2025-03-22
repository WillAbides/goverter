package goverter

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/dave/jennifer/jen"
)

type paramType int

const (
	paramsRequired paramType = iota
	paramsOptional
	paramsNone
)

const (
	argUseSource      = "source"
	argUseMultiSource = "additional-source"
	argUseInterface   = "interface"
	argUseContext     = "context"
	argUseTarget      = "target"
)

type rawArg struct {
	Name string
	Use  string
	Type *xType
}

type parameters struct {
	TypeParams bool

	Source       *xType
	MultiSources []*xType
	Target       *xType
	Context      map[string]*xType

	Signature signature

	RawArgs []rawArg

	ReturnError  bool
	UpdateTarget bool
}

type methodDefinition struct {
	parameters
	OriginID string
	Call     *jen.Statement
	ID       string
	Package  string
	Name     string

	Generated  bool
	CustomCall *jen.Statement
}

func (def *methodDefinition) ArgDebug(indent string) string {
	var lines []string
	for _, arg := range def.RawArgs {
		argUse := arg.Use
		if arg.Use == argUseMultiSource {
			argUse = argUseSource
		} else if arg.Use == argUseInterface {
			argUse = argUseContext
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

type parseMethodOpts struct {
	Location          string
	Converter         types.Type
	OutputPackagePath string

	ErrorPrefix       string
	Params            paramType
	ParamsMultiSource bool
	AllowTypeParams   bool

	ContextMatch *regexp.Regexp

	Generated   bool
	CustomCall  *jen.Statement
	UpdateParam string
}

type localMethodOpts struct {
	Context map[string]bool
}

var emptyLocalMethodOpts = localMethodOpts{Context: map[string]bool{}}

// parseMethod parses an function into a methodDefinition.
func parseMethod(obj types.Object, opts *parseMethodOpts, localOpts localMethodOpts) (*methodDefinition, error) {
	methodDef := &methodDefinition{
		ID:         obj.String(),
		OriginID:   obj.String(),
		Generated:  opts.Generated,
		CustomCall: opts.CustomCall,
		parameters: parameters{
			Context: make(map[string]*xType, 0),
		},
		Name: obj.Name(),
	}

	formatErr := func(s string) error {
		loc := ""
		if opts.Location != "" {
			loc = opts.Location + "\n    "
		}
		return fmt.Errorf("%s:\n    %s%s%s\n\n%s", opts.ErrorPrefix, loc, obj.String(), methodDef.ArgDebug("        "), s)
	}

	if !accessible(obj, opts.OutputPackagePath) {
		return nil, formatErr("must be exported")
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return nil, formatErr("must be a function")
	}
	resultsLen := sig.Results().Len()

	methodDef.TypeParams = sig.TypeParams().Len() > 0

	if pkg := obj.Pkg(); pkg != nil {
		methodDef.Package = pkg.Path()
	}

	for i := 0; i < sig.Params().Len(); i++ {
		arg := rawArg{
			Name: sig.Params().At(i).Name(),
			Type: typeOf(sig.Params().At(i).Type()),
		}

		switch {
		case types.Identical(arg.Type.T, opts.Converter):
			arg.Use = argUseInterface
		case opts.UpdateParam != "" && arg.Name == opts.UpdateParam:
			arg.Use = argUseTarget
			methodDef.Target = arg.Type
			methodDef.UpdateTarget = true

			switch {
			case resultsLen == 0:
				// okay nothing more
			case resultsLen == 1 && isError(sig.Results().At(0)):
				methodDef.ReturnError = true
			default:
				return nil, formatErr("The signature one non 'error' result or multiple results is not supported for goverter:update signatures.")
			}
		case (opts.ContextMatch != nil && opts.ContextMatch.MatchString(arg.Name)) || localOpts.Context[arg.Name]:
			methodDef.Context[arg.Type.String] = arg.Type
			arg.Use = argUseContext
		case methodDef.Source == nil:
			arg.Use = argUseSource
			methodDef.Source = arg.Type
			methodDef.Signature.Source = methodDef.Source.String
		default:
			arg.Use = argUseMultiSource
			methodDef.MultiSources = append(methodDef.MultiSources, arg.Type)
		}

		methodDef.RawArgs = append(methodDef.RawArgs, arg)
	}
	if !methodDef.UpdateTarget && opts.UpdateParam != "" {
		return nil, formatErr(fmt.Sprintf("Argument %q must exist when using 'goverter:target %s'", opts.UpdateParam, opts.UpdateParam))
	}

	if !methodDef.UpdateTarget {
		if resultsLen == 0 || resultsLen > 2 {
			return nil, formatErr("must have one or two returns")
		}
		if resultsLen == 2 {
			if isError(sig.Results().At(1)) {
				methodDef.ReturnError = true
			} else {
				return nil, formatErr("must have type error as second return but has: " + sig.Results().At(1).Type().String())
			}
		}

		methodDef.Target = typeOf(sig.Results().At(0).Type())
	}

	if methodDef.TypeParams && !opts.AllowTypeParams {
		return nil, formatErr("must not be generic")
	}

	switch {
	case opts.Params == paramsNone && methodDef.Source != nil:
		return nil, formatErr("must have no source params")
	case opts.Params == paramsRequired && methodDef.Source == nil:
		return nil, formatErr("must have at least one source param")
	case !opts.ParamsMultiSource && len(methodDef.MultiSources) > 0:
		return nil, formatErr("must have only one source param")
	}

	methodDef.Signature.Target = methodDef.Target.String

	return methodDef, nil
}

func isError(obj *types.Var) bool {
	t, ok := obj.Type().(*types.Named)
	return ok && t.Obj().Name() == "error" && t.Obj().Pkg() == nil
}
