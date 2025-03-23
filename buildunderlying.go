package goverter

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

// useUnderlyingTypeMethods handles useUnderlyingTypeMethods.
type useUnderlyingTypeMethods struct{}

// Matches returns true, if the builder can create handle the given types.
func (*useUnderlyingTypeMethods) matches(ctx *methodContext, source, target *xType) bool {
	if !ctx.Conf.UseUnderlyingTypeMethods {
		return false
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)
	return sourceUnderlying || targetUnderlying
}

// Build creates conversion source code for the given source and target type.
func (*useUnderlyingTypeMethods) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *jenID, *buildError) {
	if isBuildEnum(ctx, source, target) {
		return nil, nil, bewBuildError(fmt.Sprintf(`The conversion between the types
    %s
    %s

does qualify for enum conversion but also match an extend method via useUnderlyingTypeMethods.
You have to disable enum or useUnderlyingTypeMethods to resolve the setting conflict.`, source.String, target.String))
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)

	innerSource := source
	innerTarget := target

	if sourceUnderlying {
		innerSource = typeOf(source.NamedType.Underlying())
		sourceID = otherID(innerSource.TypeAsJen().Call(sourceID.Code))
	}

	if targetUnderlying {
		innerTarget = typeOf(target.NamedType.Underlying())
	}

	stmt, id, err := gen.Build(ctx, sourceID, innerSource, innerTarget, errPath)
	if err != nil {
		return nil, nil, err.Lift(&errorMessagePath{
			SourceID:   "*",
			SourceType: innerSource.String,
			TargetID:   "*",
			TargetType: innerTarget.String,
		})
	}

	if targetUnderlying {
		id = otherID(target.TypeAsJen().Call(id.Code))
	}

	return stmt, id, err
}

func (u *useUnderlyingTypeMethods) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	return assignByBuild(u, gen, ctx, assignTo, sourceID, source, target, errPath)
}

func findUnderlyingExtendMapping(ctx *methodContext, source, target *xType) (underlyingSource, underlyingTarget bool) {
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
