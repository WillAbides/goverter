package builder

import (
	"go/types"

	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
	"github.com/jmattheis/goverter/config"
	"github.com/jmattheis/goverter/xtype/method"
)

// ThisVar is used as name for the reference to the converter interface.
const ThisVar = "c"

// Builder builds converter implementations, and can decide if it can handle the given type.
type Builder interface {
	// Matches returns true, if the builder can create handle the given types.
	Matches(ctx *MethodContext, source, target *goverter.Type) bool

	// Build creates conversion source code for the given source and target type.
	Build(gen Generator,
		ctx *MethodContext,
		sourceID *goverter.JenID,
		source, target *goverter.Type,
		path ErrorPath,
	) ([]jen.Code, *goverter.JenID, *Error)

	// Assign creates conversion source code for the given source and target type and assigns it.
	Assign(gen Generator,
		ctx *MethodContext,
		assignTo *AssignTo,
		sourceID *goverter.JenID,
		source, target *goverter.Type,
		path ErrorPath,
	) ([]jen.Code, *Error)
}

// Generator checks all existing builders if they can create a conversion implementations for the given source and target type
// If no one Builder#Matches then, an error is returned.
type Generator interface {
	Build(
		ctx *MethodContext,
		sourceID *goverter.JenID,
		source, target *goverter.Type,
		path ErrorPath,
	) ([]jen.Code, *goverter.JenID, *Error)

	Assign(ctx *MethodContext,
		assignTo *AssignTo,
		sourceID *goverter.JenID,
		source, target *goverter.Type,
		path ErrorPath,
	) ([]jen.Code, *Error)

	CallMethod(
		ctx *MethodContext,
		method *method.Definition,
		sourceID *goverter.JenID,
		source, target *goverter.Type,
		path ErrorPath,
	) ([]jen.Code, *goverter.JenID, *Error)

	ReturnError(ctx *MethodContext,
		path ErrorPath,
		id *jen.Statement) (jen.Code, bool)
}

// MethodContext exposes information for the current method.
type MethodContext struct {
	*Namer
	Conf              *config.Method
	FieldsTarget      string
	OutputPackagePath string
	UseConstructor    bool
	Signature         goverter.Signature
	TargetType        *goverter.Type
	HasMethod         func(*MethodContext, types.Type, types.Type) bool
	SeenNamed         map[string]struct{}

	IndexID method.IndexID
	Context map[string]*goverter.JenID

	AvailableContext map[string]*goverter.Type

	TargetVar *jen.Statement
}

func (ctx *MethodContext) HasSeen(source *goverter.Type) bool {
	if !source.Named {
		return false
	}
	typeString := source.NamedType.String()
	_, ok := ctx.SeenNamed[typeString]
	return ok
}

func (ctx *MethodContext) MarkSeen(source *goverter.Type) {
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

func (ctx *MethodContext) Field(target *goverter.Type, name string) *config.FieldMapping {
	if ctx.FieldsTarget != target.String {
		return emptyMapping
	}

	prop, ok := ctx.Conf.Fields[name]
	if !ok {
		return emptyMapping
	}
	return prop
}

func (ctx *MethodContext) DefinedFields(target *goverter.Type) map[string]struct{} {
	if ctx.FieldsTarget != target.String {
		return emptyFields
	}

	f := map[string]struct{}{}
	for name := range ctx.Conf.Fields {
		f[name] = struct{}{}
	}
	return f
}

func (ctx *MethodContext) DefinedEnumFields(target *goverter.Type) map[string]struct{} {
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
	emptyMapping = &config.FieldMapping{}
	emptyFields  = map[string]struct{}{}
)
