package goverter

import (
	"github.com/dave/jennifer/jen"
)

// basic handles basic data types.
type basic struct{}

// Matches returns true, if the builder can create handle the given types.
func (*basic) matches(_ *methodContext, source, target *xType) bool {
	return source.Basic && target.Basic &&
		source.BasicType.Kind() == target.BasicType.Kind()
}

// Build creates conversion source code for the given source and target type.
func (*basic) build(
	_ *generator,
	_ *methodContext,
	sourceID *jenID,
	source, target *xType,
	_ errorPath,
) ([]jen.Code, *jenID, *buildError) {
	if target.Named || (!target.Named && source.Named) {
		return nil, otherID(target.TypeAsJen().Call(sourceID.Code)), nil
	}
	return nil, sourceID, nil
}

func (b *basic) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	return assignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}

// basicTargetPointerRule handles edge conditions if the target type is a pointer.
type basicTargetPointerRule struct{}

// Matches returns true, if the builder can create handle the given types.
func (*basicTargetPointerRule) matches(_ *methodContext, source, target *xType) bool {
	return source.Basic && target.Pointer && target.PointerInner.Basic
}

// Build creates conversion source code for the given source and target type.
func (*basicTargetPointerRule) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *jenID, *buildError) {
	name := ctx.Name(target.ID())
	ctx.SetErrorTargetVar(jen.Nil())

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, errPath)
	if err != nil {
		return nil, nil, err.Lift(&errorMessagePath{
			SourceID:   "*",
			SourceType: source.String,
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}
	stmt = append(stmt, jen.Id(name).Op(":=").Add(id.Code))
	newID := jen.Op("&").Id(name)

	return stmt, otherID(newID), err
}

func (b *basicTargetPointerRule) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	return assignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}
