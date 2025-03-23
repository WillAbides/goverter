package goverter

import (
	"fmt"
	"go/constant"
	"go/types"
	"strings"

	"github.com/dave/jennifer/jen"
)

// accessible checks if obj is accessible within outputPackagePath.
func accessible(obj types.Object, outputPackagePath string) bool {
	if obj.Exported() {
		return true
	}

	pkg := obj.Pkg()
	return pkg == nil || pkg.Path() == outputPackagePath
}

// signature represents a signature for conversion.
type signature struct {
	Source string
	Target string
}

// Type is a helper wrapper for types.Type.
type xType struct {
	String        string
	T             types.Type
	Interface     bool
	InterfaceType *types.Interface
	Struct        bool
	StructType    *types.Struct
	Named         bool
	NamedType     *types.Named
	Pointer       bool
	PointerType   *types.Pointer
	PointerInner  *xType
	List          bool
	ListFixed     bool
	ListInner     *xType
	Map           bool
	MapType       *types.Map
	MapKey        *xType
	MapValue      *xType
	Basic         bool
	BasicType     *types.Basic
	Signature     bool
	SignatureType *types.Signature
	Func          bool
	FuncType      *types.Func
	Chan          bool
	ChanType      *types.Chan

	enum *Enum
}

func (t *xType) AssignableTo(other *xType) bool {
	return types.AssignableTo(t.T, other.T)
}

func (t *xType) AsPointer() *xType {
	return typeOf(t.AsPointerType())
}

func (t *xType) AsPointerType() *types.Pointer {
	return types.NewPointer(t.T)
}

func (t *xType) inStruct(source *xType, field string) *xType {
	if t.Signature && source.Named {
		t.FuncType = types.NewFunc(-1, source.NamedType.Obj().Pkg(), field, t.SignatureType)
		t.Func = true
	}

	return t
}

// structField holds the type of a struct field and its name.
type structField struct {
	Path []string
	Type *xType
}

type simpleStructField struct {
	Name string
	Type *xType
}

func (t *xType) findAllFields(path []string, name string, ignoreCase bool) (*structField, []*structField) {
	if !t.Struct {
		panic("trying to get field of non struct")
	}

	var matches []*structField
	handle := func(obj types.Object) *structField {
		exact := obj.Name() == name
		if exact || (ignoreCase && strings.EqualFold(obj.Name(), name)) {
			// exact match takes precedence over case-insensitive match
			newPath := append([]string{}, path...)
			newPath = append(newPath, obj.Name())
			f := &structField{Path: newPath, Type: typeOf(obj.Type()).inStruct(t, obj.Name())}
			if exact {
				return f
			}
			matches = append(matches, f)
		}
		return nil
	}

	for y := 0; y < t.StructType.NumFields(); y++ {
		if exact := handle(t.StructType.Field(y)); exact != nil {
			return exact, matches
		}
	}

	if t.Named {
		for y := 0; y < t.NamedType.NumMethods(); y++ {
			if exact := handle(t.NamedType.Method(y)); exact != nil {
				return exact, matches
			}
		}
	}

	return nil, matches
}

type fieldSources struct {
	Path []string
	Type *xType
}

func findExactField(source *xType, name string) (*simpleStructField, error) {
	exactMatch, _ := source.findAllFields(nil, name, false)
	if exactMatch == nil {
		return nil, fmt.Errorf("%q does not exist", name)
	}
	return &simpleStructField{Name: exactMatch.Path[0], Type: exactMatch.Type}, nil
}

type noMatchError struct{ Field string }

func (err *noMatchError) Error() string {
	return fmt.Sprintf("\"%s\" does not exist", err.Field)
}

func findField(
	name string,
	ignoreCase bool,
	source *xType,
	additionalFieldSources []fieldSources,
) (*structField, error) {
	exactMatch, ignoreCaseMatches := source.findAllFields(nil, name, ignoreCase)
	var exactMatches []*structField
	if exactMatch != nil {
		exactMatches = append(exactMatches, exactMatch)
	}

	for _, source := range additionalFieldSources {
		sourceExactMatch, sourceIgnoreCaseMatches := source.Type.findAllFields(source.Path, name, ignoreCase)
		if sourceExactMatch != nil {
			exactMatches = append(exactMatches, sourceExactMatch)
		}
		ignoreCaseMatches = append(ignoreCaseMatches, sourceIgnoreCaseMatches...)
	}

	matches := exactMatches
	if len(matches) == 0 {
		matches = ignoreCaseMatches
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, &noMatchError{Field: name}
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, strings.Join(m.Path, "."))
		}
		return nil, ambiguousMatchError(name, names)
	}
}

// JenID a jennifer code wrapper with extra infos.
type JenID struct {
	ParentPointer *JenID
	Code          *jen.Statement
	Variable      bool
}

func (j *JenID) Pointer(t *xType, namer func(string) string) ([]jen.Code, *JenID) {
	if j.Variable {
		return nil, otherID(jen.Op("&").Add(j.Code.Clone()))
	}

	name := namer(t.ID())
	stmt := []jen.Code{jen.Id(name).Op(":=").Add(j.Code.Clone())}
	return stmt, otherID(jen.Op("&").Id(name))
}

func (j *JenID) Deref(source *xType) *JenID {
	valueSourceID := jen.Op("*").Add(j.Code.Clone())
	if !source.PointerInner.Basic {
		valueSourceID = jen.Parens(valueSourceID)
	}
	innerID := otherID(valueSourceID)
	innerID.ParentPointer = j
	return innerID
}

// variableID is used, when the ID can be referenced. F.ex it is not a function call.
func variableID(code *jen.Statement) *JenID {
	return &JenID{Code: code, Variable: true}
}

// otherID is used, when the ID isn't a variable id.
func otherID(code *jen.Statement) *JenID {
	return &JenID{Code: code, Variable: false}
}

// typeOf creates a Type.
func typeOf(t types.Type) *xType {
	t = types.Unalias(t)
	rt := &xType{}
	rt.T = t
	rt.String = t.String()
	applyTo(rt, t)
	return rt
}

func applyTo(rt *xType, t types.Type) {
	switch value := t.(type) {
	case *types.Pointer:
		rt.Pointer = true
		rt.PointerType = value
		rt.PointerInner = typeOf(value.Elem())
	case *types.Basic:
		rt.Basic = true
		rt.BasicType = value
	case *types.Map:
		rt.Map = true
		rt.MapType = value
		rt.MapKey = typeOf(value.Key())
		rt.MapValue = typeOf(value.Elem())
	case *types.Slice:
		rt.List = true
		rt.ListInner = typeOf(value.Elem())
	case *types.Array:
		rt.List = true
		rt.ListFixed = true
		rt.ListInner = typeOf(value.Elem())
	case *types.Named:
		rt.Named = true
		rt.NamedType = value
		applyTo(rt, value.Underlying())
	case *types.Struct:
		rt.Struct = true
		rt.StructType = value
	case *types.Interface:
		rt.Interface = true
		rt.InterfaceType = value
	case *types.Signature:
		rt.Signature = true
		rt.SignatureType = value
	case *types.Chan:
		rt.Chan = true
		rt.ChanType = value
	case *types.TypeParam:
		// ignore
	default:
		panic("unknown types.Type " + t.String())
	}
}

// ID returns a deteministically generated id that may be used as variable.
func (t *xType) ID() string {
	return t.asID(true, true)
}

// UnescapedID returns a deteministically generated id that may be used as variable
// reserved keywords aren't escaped.
func (t *xType) UnescapedID() string {
	return t.asID(true, false)
}

func (t *xType) asID(seeNamed, escapeReserved bool) string {
	if seeNamed && t.Named {
		pkg := t.NamedType.Obj().Pkg()
		name := t.NamedType.Obj().Name()
		switch {
		case pkg != nil:
			name = pkg.Name() + name
		case escapeReserved:
			name = "x" + name
		}
		return name
	}
	if t.List {
		return t.ListInner.asID(true, false) + "List"
	}
	if t.Basic {
		if escapeReserved {
			return "x" + t.BasicType.String()
		}
		return t.BasicType.String()
	}
	if t.Pointer {
		return "p" + strings.Title(t.PointerInner.asID(true, false))
	}
	if t.Map {
		return "map" + strings.Title(t.MapKey.asID(true, false)+strings.Title(t.MapValue.asID(true, false)))
	}
	if t.Struct {
		return "unnamed"
	}
	if t.Chan {
		return "chan"
	}
	return "unknown"
}

// TypeAsJen returns a jen representation of the type.
func (t *xType) TypeAsJen() *jen.Statement {
	if t.Named {
		return toCode(t.NamedType)
	}
	return toCode(t.T)
}

func ambiguousMatchError(name string, ambNames []string) error {
	return fmt.Errorf(`multiple matches found for %q. Possible matches: %s.

Explicitly define the mapping via goverter:map. Example:

    goverter:map %s %s

See https://goverter.jmattheis.de/reference/map`, name, strings.Join(ambNames, ", "), ambNames[0], name)
}

func (t *xType) Enum(cfg *enumConfig) *Enum {
	if !t.Named {
		return &Enum{}
	}

	if t.enum == nil {
		t.enum = loadEnum(t.NamedType, cfg)
	}
	return t.enum
}

func loadEnum(t *types.Named, cfg *enumConfig) *Enum {
	path := t.Obj().Pkg().Path()
	name := t.Obj().Name()

	if !cfg.enabled || cfg.excludes.Matches(path, name) {
		return &Enum{}
	}

	e := detectEnum(t)
	return &e
}

func detectEnum(named *types.Named) Enum {
	basic, ok := named.Underlying().(*types.Basic)
	if !ok {
		return Enum{}
	}

	if basic.Info()&(types.IsFloat|types.IsString|types.IsInteger) == 0 {
		return Enum{}
	}

	scope := named.Obj().Pkg().Scope()

	members := map[string]any{}
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok {
			continue
		}

		if types.Identical(named, c.Type()) {
			members[name] = constant.Val(c.Val())
		}
	}

	if len(members) == 0 {
		return Enum{}
	}

	return Enum{
		Type:    named,
		Members: members,
		OK:      true,
	}
}
