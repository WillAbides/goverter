package builder

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

// Basic handles basic data types.
type Basic struct{}

// Matches returns true, if the builder can create handle the given types.
func (*Basic) Matches(_ *MethodContext, source, target *goverter.Type) bool {
	return source.Basic && target.Basic &&
		source.BasicType.Kind() == target.BasicType.Kind()
}

// Build creates conversion source code for the given source and target type.
func (*Basic) Build(_ Generator, _ *MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	if target.Named || (!target.Named && source.Named) {
		return nil, goverter.OtherID(target.TypeAsJen().Call(sourceID.Code)), nil
	}
	return nil, sourceID, nil
}

func (b *Basic) Assign(gen Generator, ctx *MethodContext, assignTo *AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return AssignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}

// BasicTargetPointerRule handles edge conditions if the target type is a pointer.
type BasicTargetPointerRule struct{}

// Matches returns true, if the builder can create handle the given types.
func (*BasicTargetPointerRule) Matches(_ *MethodContext, source, target *goverter.Type) bool {
	return source.Basic && target.Pointer && target.PointerInner.Basic
}

// Build creates conversion source code for the given source and target type.
func (*BasicTargetPointerRule) Build(gen Generator, ctx *MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	name := ctx.Name(target.ID())
	ctx.SetErrorTargetVar(jen.Nil())

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, errPath)
	if err != nil {
		return nil, nil, err.Lift(&goverter.ErrorMessagePath{
			SourceID:   "*",
			SourceType: source.String,
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}
	stmt = append(stmt, jen.Id(name).Op(":=").Add(id.Code))
	newID := jen.Op("&").Id(name)

	return stmt, goverter.OtherID(newID), err
}

func (b *BasicTargetPointerRule) Assign(gen Generator, ctx *MethodContext, assignTo *AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return AssignByBuild(b, gen, ctx, assignTo, sourceID, source, target, errPath)
}
