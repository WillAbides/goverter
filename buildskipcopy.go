package goverter

import (
	"github.com/dave/jennifer/jen"
)

// skipCopy handles FlagSkipCopySameType.
type skipCopy struct{}

// Matches returns true, if the builder can create handle the given types.
func (*skipCopy) matches(ctx *methodContext, source, target *xType) bool {
	return ctx.Conf.SkipCopySameType && source.String == target.String
}

// Build creates conversion source code for the given source and target type.
func (*skipCopy) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *jenID, *buildError) {
	return nil, sourceID, nil
}

func (*skipCopy) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *buildError) {
	return []jen.Code{assignTo.Stmt.Clone().Op("=").Add(sourceID.Code)}, nil
}
