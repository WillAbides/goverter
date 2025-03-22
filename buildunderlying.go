package goverter

import (
	"fmt"

	"github.com/dave/jennifer/jen"
)

// UseUnderlyingTypeMethods handles UseUnderlyingTypeMethods.
type UseUnderlyingTypeMethods struct{}

// Matches returns true, if the builder can create handle the given types.
func (*UseUnderlyingTypeMethods) matches(ctx *MethodContext, source, target *Type) bool {
	if !ctx.Conf.UseUnderlyingTypeMethods {
		return false
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)
	return sourceUnderlying || targetUnderlying
}

// Build creates conversion source code for the given source and target type.
func (*UseUnderlyingTypeMethods) build(
	gen Generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	if IsBuildEnum(ctx, source, target) {
		return nil, nil, NewBuildError(fmt.Sprintf(`The conversion between the types
    %s
    %s

does qualify for enum conversion but also match an extend method via useUnderlyingTypeMethods.
You have to disable enum or useUnderlyingTypeMethods to resolve the setting conflict.`, source.String, target.String))
	}

	sourceUnderlying, targetUnderlying := findUnderlyingExtendMapping(ctx, source, target)

	innerSource := source
	innerTarget := target

	if sourceUnderlying {
		innerSource = TypeOf(source.NamedType.Underlying())
		sourceID = OtherID(innerSource.TypeAsJen().Call(sourceID.Code))
	}

	if targetUnderlying {
		innerTarget = TypeOf(target.NamedType.Underlying())
	}

	stmt, id, err := gen.Build(ctx, sourceID, innerSource, innerTarget, errPath)
	if err != nil {
		return nil, nil, err.Lift(&ErrorMessagePath{
			SourceID:   "*",
			SourceType: innerSource.String,
			TargetID:   "*",
			TargetType: innerTarget.String,
		})
	}

	if targetUnderlying {
		id = OtherID(target.TypeAsJen().Call(id.Code))
	}

	return stmt, id, err
}

func (u *UseUnderlyingTypeMethods) assign(
	gen Generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	return assignByBuild(u, gen, ctx, assignTo, sourceID, source, target, errPath)
}

func findUnderlyingExtendMapping(ctx *MethodContext, source, target *Type) (underlyingSource, underlyingTarget bool) {
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
