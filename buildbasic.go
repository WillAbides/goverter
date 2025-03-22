package goverter

import (
	"github.com/dave/jennifer/jen"
)

// basic handles basic data types.
type basic struct{}

// Matches returns true, if the builder can create handle the given types.
func (*basic) matches(_ *MethodContext, source, target *Type) bool {
	return source.Basic && target.Basic &&
		source.BasicType.Kind() == target.BasicType.Kind()
}

// Build creates conversion source code for the given source and target type.
func (*basic) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	path ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	if target.Named || (!target.Named && source.Named) {
		return nil, OtherID(target.TypeAsJen().Call(sourceID.Code)), nil
	}
	return nil, sourceID, nil
}

func (b *basic) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	return assignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}

// BasicTargetPointerRule handles edge conditions if the target type is a pointer.
type BasicTargetPointerRule struct{}

// Matches returns true, if the builder can create handle the given types.
func (*BasicTargetPointerRule) matches(_ *MethodContext, source, target *Type) bool {
	return source.Basic && target.Pointer && target.PointerInner.Basic
}

// Build creates conversion source code for the given source and target type.
func (*BasicTargetPointerRule) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	name := ctx.Name(target.ID())
	ctx.SetErrorTargetVar(jen.Nil())

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, errPath)
	if err != nil {
		return nil, nil, err.Lift(&ErrorMessagePath{
			SourceID:   "*",
			SourceType: source.String,
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}
	stmt = append(stmt, jen.Id(name).Op(":=").Add(id.Code))
	newID := jen.Op("&").Id(name)

	return stmt, OtherID(newID), err
}

func (b *BasicTargetPointerRule) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	return assignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}
