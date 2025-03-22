package goverter

import (
	"go/types"

	"github.com/dave/jennifer/jen"
)

// thisVar is used as name for the reference to the converter interface.
const thisVar = "c"

// MethodContext exposes information for the current method.
type MethodContext struct {
	*Namer
	Conf              *method
	FieldsTarget      string
	OutputPackagePath string
	UseConstructor    bool
	Signature         signature
	TargetType        *xType
	HasMethod         func(*MethodContext, types.Type, types.Type) bool
	SeenNamed         map[string]struct{}

	IndexID methodIndexID
	Context map[string]*JenID

	AvailableContext map[string]*xType

	TargetVar *jen.Statement
}

func (ctx *MethodContext) HasSeen(source *xType) bool {
	if !source.Named {
		return false
	}
	typeString := source.NamedType.String()
	_, ok := ctx.SeenNamed[typeString]
	return ok
}

func (ctx *MethodContext) MarkSeen(source *xType) {
	if !source.Named {
		return
	}
	typeString := source.NamedType.String()
	ctx.SeenNamed[typeString] = struct{}{}
}

func (ctx *MethodContext) SetErrorTargetVar(m *jen.Statement) {
	if ctx.TargetVar == nil {
		ctx.TargetVar = m
	}
}

func (ctx *MethodContext) Field(target *xType, name string) *fieldMapping {
	if ctx.FieldsTarget != target.String {
		return emptyMapping
	}

	prop, ok := ctx.Conf.Fields[name]
	if !ok {
		return emptyMapping
	}
	return prop
}

func (ctx *MethodContext) DefinedFields(target *xType) map[string]struct{} {
	if ctx.FieldsTarget != target.String {
		return emptyFields
	}

	f := map[string]struct{}{}
	for name := range ctx.Conf.Fields {
		f[name] = struct{}{}
	}
	return f
}

func (ctx *MethodContext) DefinedEnumFields(target *xType) map[string]struct{} {
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
	matches(ctx *MethodContext, source, target *xType) bool

	// build creates conversion source code for the given source and target type.
	build(
		gen *generator,
		ctx *MethodContext,
		sourceID *JenID,
		source, target *xType,
		path ErrorPath,
	) ([]jen.Code, *JenID, *BuildError)

	// assign creates conversion source code for the given source and target type and assigns it.
	assign(
		gen *generator,
		ctx *MethodContext,
		assignTo *AssignTo,
		sourceID *JenID,
		source, target *xType,
		path ErrorPath,
	) ([]jen.Code, *BuildError)
}

func buildTargetVar(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *xType,
	errPath ErrorPath,
) ([]jen.Code, *jen.Statement, *BuildError) {
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

type AssignTo struct {
	Stmt   *jen.Statement
	Must   bool
	Update bool
}

func AssignOf(s *jen.Statement) *AssignTo {
	return &AssignTo{Stmt: s}
}

func (a *AssignTo) WithIndex(s *jen.Statement) *AssignTo {
	return &AssignTo{
		Stmt: a.Stmt.Clone().Index(s),
	}
}

func (a *AssignTo) MustAssign() *AssignTo {
	a.Must = true
	return a
}

func (a *AssignTo) IsUpdate() *AssignTo {
	a.Update = true
	return a
}

func ToAssignable(assignTo *AssignTo) func(
	stmt []jen.Code,
	nextID *JenID,
	err *BuildError,
) ([]jen.Code, *BuildError) {
	return func(stmt []jen.Code, nextID *JenID, err *BuildError) ([]jen.Code, *BuildError) {
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
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *xType,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	return ToAssignable(assignTo)(b.build(gen, ctx, sourceID, source, target, errPath))
}

func buildByAssign(
	b builder,
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *xType,
	path ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
	if err != nil {
		return nil, nil, err
	}

	stmt, err := b.assign(gen, ctx, AssignOf(valueVar), sourceID, source, target, path)
	if err != nil {
		return nil, nil, err
	}

	buildStmt = append(buildStmt, stmt...)
	return buildStmt, variableID(valueVar), nil
}
