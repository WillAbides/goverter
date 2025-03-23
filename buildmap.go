package goverter

import (
	"github.com/dave/jennifer/jen"
)

// buildMap handles map types.
type buildMap struct{}

// Matches returns true, if the builder can create handle the given types.
func (*buildMap) matches(_ *methodContext, source, target *xType) bool {
	return source.Map && target.Map
}

// Build creates conversion source code for the given source and target type.
func (m *buildMap) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *jenID, *buildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	return buildByAssign(m, gen, ctx, sourceID, source, target, errPath)
}

func (*buildMap) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	key, value := ctx.Map()

	errPath = errPath.Key(jen.Id(key))

	block, keyID, err := gen.Build(ctx, variableID(jen.Id(key)), source.MapKey, target.MapKey, errPath)
	if err != nil {
		return nil, err.Lift(&errorMessagePath{
			SourceID:   "[]",
			SourceType: "<mapkey> " + source.MapKey.String,
			TargetID:   "[]",
			TargetType: "<mapkey> " + target.MapKey.String,
		})
	}
	valueStmt, err := gen.Assign(
		ctx, assignTo.WithIndex(keyID.Code).MustAssign(), variableID(jen.Id(value)), source.MapValue, target.MapValue, errPath)
	if err != nil {
		return nil, err.Lift(&errorMessagePath{
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
