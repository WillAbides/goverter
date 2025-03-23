package goverter

import (
	"go/types"

	"github.com/dave/jennifer/jen"
)

// thisVar is used as name for the reference to the converter interface.
const thisVar = "c"

// methodContext exposes information for the current method.
type methodContext struct {
	*namer
	Conf              *method
	FieldsTarget      string
	OutputPackagePath string
	UseConstructor    bool
	Signature         signature
	TargetType        *xType
	HasMethod         func(*methodContext, types.Type, types.Type) bool
	SeenNamed         map[string]struct{}

	IndexID methodIndexID
	Context map[string]*jenID

	AvailableContext map[string]*xType

	TargetVar *jen.Statement
}

func (ctx *methodContext) HasSeen(source *xType) bool {
	if !source.Named {
		return false
	}
	typeString := source.NamedType.String()
	_, ok := ctx.SeenNamed[typeString]
	return ok
}

func (ctx *methodContext) MarkSeen(source *xType) {
	if !source.Named {
		return
	}
	typeString := source.NamedType.String()
	ctx.SeenNamed[typeString] = struct{}{}
}

func (ctx *methodContext) SetErrorTargetVar(m *jen.Statement) {
	if ctx.TargetVar == nil {
		ctx.TargetVar = m
	}
}

func (ctx *methodContext) Field(target *xType, name string) *fieldMapping {
	if ctx.FieldsTarget != target.String {
		return emptyMapping
	}

	prop, ok := ctx.Conf.Fields[name]
	if !ok {
		return emptyMapping
	}
	return prop
}

func (ctx *methodContext) DefinedFields(target *xType) map[string]struct{} {
	if ctx.FieldsTarget != target.String {
		return emptyFields
	}

	f := map[string]struct{}{}
	for name := range ctx.Conf.Fields {
		f[name] = struct{}{}
	}
	return f
}

func (ctx *methodContext) DefinedEnumFields(target *xType) map[string]struct{} {
	if ctx.FieldsTarget != target.String {
		return emptyFields
	}

	f := map[string]struct{}{}
	for name := range ctx.Conf.EnumMapping.Map {
		f[name] = struct{}{}
	}
	return f
}

var (
	emptyMapping = &fieldMapping{}
	emptyFields  = map[string]struct{}{}
)

// builder builds converter implementations, and can decide if it can handle the given type.
type builder interface {
	// matches returns true, if the builder can create handle the given types.
	matches(ctx *methodContext, source, target *xType) bool

	// build creates conversion source code for the given source and target type.
	build(
		gen *generator,
		ctx *methodContext,
		sourceID *jenID,
		source, target *xType,
		path errorPath,
	) ([]jen.Code, *jenID, *buildError)

	// assign creates conversion source code for the given source and target type and assigns it.
	assign(
		gen *generator,
		ctx *methodContext,
		assignTo *assignTo,
		sourceID *jenID,
		source, target *xType,
		path errorPath,
	) ([]jen.Code, *buildError)
}

func buildTargetVar(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *jen.Statement, *buildError) {
	if !ctx.UseConstructor ||
		!types.Identical(ctx.Conf.Source.T, source.T) ||
		!types.Identical(ctx.Conf.Target.T, target.T) {
		name := ctx.Name(target.ID())
		variable := jen.Var().Id(name).Add(target.TypeAsJen())
		ctx.SetErrorTargetVar(jen.Id(name))
		return []jen.Code{variable}, jen.Id(name), nil
	}
	ctx.UseConstructor = false

	callTarget := target
	toPointer := target.Pointer && !ctx.Conf.Constructor.Target.Pointer
	if toPointer {
		callTarget = target.PointerInner
	}

	stmt, nextID, err := gen.CallMethod(ctx, ctx.Conf.Constructor, sourceID, source, callTarget, errPath)
	if err != nil {
		return nil, nil, err
	}

	if toPointer {
		pstmt, pointerID := nextID.Pointer(callTarget, ctx.Name)
		stmt = append(stmt, pstmt...)
		nextID = pointerID
	}

	if nextID.Variable {
		ctx.SetErrorTargetVar(nextID.Code.Clone())
		return stmt, nextID.Code, nil
	}
	name := ctx.Name(target.ID())
	stmt = append(stmt, jen.Id(name).Op(":=").Add(nextID.Code))
	ctx.SetErrorTargetVar(jen.Id(name))
	return stmt, jen.Id(name), nil
}

type assignTo struct {
	Stmt   *jen.Statement
	Must   bool
	Update bool
}

func assignOf(s *jen.Statement) *assignTo {
	return &assignTo{Stmt: s}
}

func (a *assignTo) WithIndex(s *jen.Statement) *assignTo {
	return &assignTo{
		Stmt: a.Stmt.Clone().Index(s),
	}
}

func (a *assignTo) MustAssign() *assignTo {
	a.Must = true
	return a
}

func (a *assignTo) IsUpdate() *assignTo {
	a.Update = true
	return a
}

func toAssignable(assignTo *assignTo) func(stmt []jen.Code, nextID *jenID, err *buildError) ([]jen.Code, *buildError) {
	return func(stmt []jen.Code, nextID *jenID, err *buildError) ([]jen.Code, *buildError) {
		if err != nil {
			return nil, err
		}
		stmt = append(stmt, assignTo.Stmt.Clone().Op("=").Add(nextID.Code))
		return stmt, nil
	}
}

func assignByBuild(
	b builder,
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	return toAssignable(assignTo)(b.build(gen, ctx, sourceID, source, target, errPath))
}

func buildByAssign(
	b builder,
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *jenID, *buildError) {
	buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
	if err != nil {
		return nil, nil, err
	}

	stmt, err := b.assign(gen, ctx, assignOf(valueVar), sourceID, source, target, path)
	if err != nil {
		return nil, nil, err
	}

	buildStmt = append(buildStmt, stmt...)
	return buildStmt, variableID(valueVar), nil
}
