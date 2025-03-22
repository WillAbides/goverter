package builder

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

// Map handles map types.
type Map struct{}

// Matches returns true, if the builder can create handle the given types.
func (*Map) Matches(_ *goverter.MethodContext, source, target *goverter.Type) bool {
	return source.Map && target.Map
}

// Build creates conversion source code for the given source and target type.
func (m *Map) Build(gen goverter.Generator, ctx *goverter.MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	return goverter.BuildByAssign(m, gen, ctx, sourceID, source, target, errPath)
}

func (*Map) Assign(gen goverter.Generator, ctx *goverter.MethodContext, assignTo *goverter.AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	key, value := ctx.Map()

	errPath = errPath.Key(jen.Id(key))

	block, keyID, err := gen.Build(ctx, goverter.VariableID(jen.Id(key)), source.MapKey, target.MapKey, errPath)
	if err != nil {
		return nil, err.Lift(&goverter.ErrorMessagePath{
			SourceID:   "[]",
			SourceType: "<mapkey> " + source.MapKey.String,
			TargetID:   "[]",
			TargetType: "<mapkey> " + target.MapKey.String,
		})
	}
	valueStmt, err := gen.Assign(
		ctx, assignTo.WithIndex(keyID.Code).MustAssign(), goverter.VariableID(jen.Id(value)), source.MapValue, target.MapValue, errPath)
	if err != nil {
		return nil, err.Lift(&goverter.ErrorMessagePath{
			SourceID:   "[]",
			SourceType: "<mapvalue> " + source.MapValue.String,
			TargetID:   "[]",
			TargetType: "<mapvalue> " + target.MapValue.String,
		})
	}
	block = append(block, valueStmt...)

	stmt := []jen.Code{
		jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(
			assignTo.Stmt.Clone().Op("=").Make(target.TypeAsJen(), jen.Len(sourceID.Code.Clone())),
			jen.For(jen.List(jen.Id(key), jen.Id(value)).Op(":=").Range().Add(sourceID.Code)).
				Block(block...),
		),
	}

	return stmt, nil
}
