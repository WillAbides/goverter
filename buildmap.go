package goverter

import (
	"github.com/dave/jennifer/jen"
)

// BuildMap handles map types.
type BuildMap struct{}

// Matches returns true, if the builder can create handle the given types.
func (*BuildMap) matches(_ *MethodContext, source, target *Type) bool {
	return source.Map && target.Map
}

// Build creates conversion source code for the given source and target type.
func (m *BuildMap) build(
	gen Generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	return buildByAssign(m, gen, ctx, sourceID, source, target, errPath)
}

func (*BuildMap) assign(
	gen Generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	key, value := ctx.Map()

	errPath = errPath.Key(jen.Id(key))

	block, keyID, err := gen.Build(ctx, VariableID(jen.Id(key)), source.MapKey, target.MapKey, errPath)
	if err != nil {
		return nil, err.Lift(&ErrorMessagePath{
			SourceID:   "[]",
			SourceType: "<mapkey> " + source.MapKey.String,
			TargetID:   "[]",
			TargetType: "<mapkey> " + target.MapKey.String,
		})
	}
	valueStmt, err := gen.Assign(
		ctx, assignTo.WithIndex(keyID.Code).MustAssign(), VariableID(jen.Id(value)), source.MapValue, target.MapValue, errPath)
	if err != nil {
		return nil, err.Lift(&ErrorMessagePath{
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
