package builder

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

// UseUnderlyingTypeMethods handles UseUnderlyingTypeMethods.
type UseUnderlyingTypeMethods struct{}

// Matches returns true, if the builder can create handle the given types.
func (*UseUnderlyingTypeMethods) Matches(ctx *goverter.MethodContext, source, target *goverter.Type) bool {
	if !ctx.Conf.UseUnderlyingTypeMethods {
		return false
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)
	return sourceUnderlying || targetUnderlying
}

// Build creates conversion source code for the given source and target type.
func (*UseUnderlyingTypeMethods) Build(gen goverter.Generator, ctx *goverter.MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	if goverter.IsBuildEnum(ctx, source, target) {
		return nil, nil, goverter.NewBuildError(fmt.Sprintf(`The conversion between the types
    %s
    %s

does qualify for enum conversion but also match an extend method via useUnderlyingTypeMethods.
You have to disable enum or useUnderlyingTypeMethods to resolve the setting conflict.`, source.String, target.String))
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)

	innerSource := source
	innerTarget := target

	if sourceUnderlying {
		innerSource = goverter.TypeOf(source.NamedType.Underlying())
		sourceID = goverter.OtherID(innerSource.TypeAsJen().Call(sourceID.Code))
	}

	if targetUnderlying {
		innerTarget = goverter.TypeOf(target.NamedType.Underlying())
	}

	stmt, id, err := gen.Build(ctx, sourceID, innerSource, innerTarget, errPath)
	if err != nil {
		return nil, nil, err.Lift(&goverter.ErrorMessagePath{
			SourceID:   "*",
			SourceType: innerSource.String,
			TargetID:   "*",
			TargetType: innerTarget.String,
		})
	}

	if targetUnderlying {
		id = goverter.OtherID(target.TypeAsJen().Call(id.Code))
	}

	return stmt, id, err
}

func (u *UseUnderlyingTypeMethods) Assign(gen goverter.Generator, ctx *goverter.MethodContext, assignTo *goverter.AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return goverter.AssignByBuild(u, gen, ctx, assignTo, sourceID, source, target, errPath)
}

func findUnderlyingExtendMapping(ctx *goverter.MethodContext, source, target *goverter.Type) (underlyingSource, underlyingTarget bool) {
	if source.Named {
		if ctx.HasMethod(ctx, source.NamedType.Underlying(), target.NamedType) {
			return true, false
		}

		if target.Named && ctx.HasMethod(ctx, source.NamedType.Underlying(), target.NamedType.Underlying()) {
			return true, true
		}
	}

	if target.Named && ctx.HasMethod(ctx, source.NamedType, target.NamedType.Underlying()) {
		return false, true
	}

	return false, false
}
