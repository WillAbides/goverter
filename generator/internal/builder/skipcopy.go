package builder

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

// SkipCopy handles FlagSkipCopySameType.
type SkipCopy struct{}

// Matches returns true, if the builder can create handle the given types.
func (*SkipCopy) Matches(ctx *MethodContext, source, target *goverter.Type) bool {
	return ctx.Conf.SkipCopySameType && source.String == target.String
}

// Build creates conversion source code for the given source and target type.
func (*SkipCopy) Build(_ Generator, _ *MethodContext, sourceID *goverter.JenID, _, _ *goverter.Type, _ goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	return nil, sourceID, nil
}

func (*SkipCopy) Assign(_ Generator, _ *MethodContext, assignTo *AssignTo, sourceID *goverter.JenID, _, _ *goverter.Type, _ goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return []jen.Code{assignTo.Stmt.Clone().Op("=").Add(sourceID.Code)}, nil
}
