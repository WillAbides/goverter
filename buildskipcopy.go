package goverter

import (
	"github.com/dave/jennifer/jen"
)

// SkipCopy handles FlagSkipCopySameType.
type SkipCopy struct{}

// Matches returns true, if the builder can create handle the given types.
func (*SkipCopy) matches(ctx *MethodContext, source, target *xType) bool {
	return ctx.Conf.SkipCopySameType && source.String == target.String
}

// Build creates conversion source code for the given source and target type.
func (*SkipCopy) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *xType,
	path ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	return nil, sourceID, nil
}

func (*SkipCopy) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *xType,
	path ErrorPath,
) ([]jen.Code, *BuildError) {
	return []jen.Code{assignTo.Stmt.Clone().Op("=").Add(sourceID.Code)}, nil
}
