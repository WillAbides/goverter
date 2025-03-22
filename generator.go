package goverter

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
)

type generatedMethod struct {
	*Method

	Explicit bool
	Dirty    bool

	OriginPath []MethodIndexID
	Jen        jen.Code

	IndexID MethodIndexID
}

type generator struct {
	namer  *Namer
	conf   *Converter
	lookup *MethodIndex[generatedMethod]
	extend *MethodIndex[MethodDefinition]
}

func (g *generator) getGenMethods() []*generatedMethod {
	genMethods := g.lookup.GetAll()
	sort.Slice(genMethods, func(i, j int) bool {
		return genMethods[i].Name < genMethods[j].Name
	})
	return genMethods
}

func (g *generator) buildMethods(f *jen.File) error {
	for g.anyDirty() {
		if err := g.buildDirtyMethods(); err != nil {
			return err
		}
	}
	g.appendGenerated(f)
	return nil
}

func (g *generator) buildDirtyMethods() error {
	for _, genMethod := range g.getGenMethods() {
		if !genMethod.Dirty {
			continue
		}
		genMethod.Dirty = false
		err := g.buildMethod(genMethod, genMethod.Context)
		if err != nil {
			err = err.Lift(&ErrorMessagePath{
				SourceID:   "source",
				TargetID:   "target",
				SourceType: genMethod.Source.String,
				TargetType: genMethod.Target.String,
			})
			return fmt.Errorf("Error while creating converter method:\n    %s\n    %s%s\n\n%s", genMethod.Location, genMethod.ID, genMethod.MethodDefinition.ArgDebug("        "), BuildErrorToString(err))
		}
	}
	return nil
}

func (g *generator) anyDirty() bool {
	for _, m := range g.getGenMethods() {
		if m.Dirty {
			return true
		}
	}
	return false
}

func (g *generator) appendGenerated(f *jen.File) {
	genMethods := g.getGenMethods()
	for _, raw := range g.conf.OutputRaw {
		f.Id(raw)
	}

	if g.conf.OutputFormat == OutputFormatStruct {
		if len(g.conf.Comments) > 0 {
			f.Comment(strings.Join(g.conf.Comments, "\n"))
		}
		f.Type().Id(g.conf.Name).Struct()
	}

	var init []jen.Code
	var funcs []jen.Code

	for _, def := range genMethods {
		switch g.conf.OutputFormat {
		case OutputFormatStruct:
			funcs = append(funcs, jen.Func().Params(jen.Id(ThisVar).Op("*").Id(g.conf.Name)).Id(def.Name).Add(def.Jen))
		case OutputFormatVariable:
			if def.Explicit {
				init = append(init, jen.Qual(def.Package, def.Name).Op("=").Func().Add(def.Jen))
			} else {
				funcs = append(funcs, jen.Func().Id(def.Name).Add(def.Jen))
			}
		case OutputFormatFunction:
			funcs = append(funcs, jen.Func().Id(def.Name).Add(def.Jen))
		}
	}

	if len(init) > 0 {
		f.Func().Id("init").Params().Block(init...)
	}

	for _, fn := range funcs {
		f.Add(fn)
	}
}

func (g *generator) buildMethod(genMethod *generatedMethod, context map[string]*Type) *BuildError {
	var sourceID *JenID
	source := genMethod.Source
	target := genMethod.Target

	fieldsTarget := genMethod.Target.String
	if genMethod.Target.Pointer && genMethod.Target.PointerInner.Struct {
		fieldsTarget = genMethod.Target.PointerInner.String
	}

	ctx := &MethodContext{
		Namer:             NewNamer(),
		Conf:              genMethod.Method,
		FieldsTarget:      fieldsTarget,
		AvailableContext:  context,
		SeenNamed:         map[string]struct{}{},
		TargetType:        genMethod.Target,
		Context:           map[string]*JenID{},
		IndexID:           genMethod.IndexID,
		Signature:         genMethod.Signature,
		HasMethod:         g.hasMethod,
		OutputPackagePath: g.conf.OutputPackagePath,
		UseConstructor:    genMethod.Constructor != nil,
	}

	var targetAssign *jen.Statement
	args := []jen.Code{}
	for _, arg := range genMethod.RawArgs {
		switch arg.Use {
		case ArgUseInterface:
			panic("hopefully unreachable")
		case ArgUseContext:
			name := ctx.Name("context")
			ctx.Context[arg.Type.String] = VariableID(jen.Id(name))
			args = append(args, jen.Id(name).Add(arg.Type.TypeAsJen()))
		case ArgUseSource:
			name := ctx.Name("source")
			sourceID = VariableID(jen.Id(name))
			args = append(args, jen.Id(name).Add(arg.Type.TypeAsJen()))
		case ArgUseTarget:
			name := ctx.Name("target")
			targetAssign = jen.Id(name)
			args = append(args, jen.Id(name).Add(arg.Type.TypeAsJen()))
		case ArgUseMultiSource:
			panic("multi source aren't supported right now. https://github.com/jmattheis/goverter/issues/143")
		}
	}

	var returns []jen.Code
	if targetAssign == nil {
		returns = append(returns, target.TypeAsJen())
	}

	if genMethod.ReturnError {
		returns = append(returns, jen.Id("error"))
	}

	var funcBlock []jen.Code
	if targetAssign != nil {
		var err *BuildError
		funcBlock, err = g.convertTo(ctx, AssignOf(targetAssign), sourceID, source, target, nil)
		if err != nil {
			return err
		}

		if genMethod.ReturnError {
			funcBlock = append(funcBlock, jen.Return().Nil())
		}
	} else if def, err := g.extend.Get(ctx.Signature, context); def != nil {
		jenReturn, err := g.delegateMethod(ctx, def, sourceID)
		if err != nil {
			return err
		}
		funcBlock = []jen.Code{jenReturn}
	} else if err != nil {
		return NewBuildError(err.Error())
	} else {
		stmt, newID, err := g.buildNoLookup(ctx, sourceID, source, target, nil)
		if err != nil {
			return err
		}
		ret := []jen.Code{newID.Code}
		if genMethod.ReturnError {
			ret = append(ret, jen.Nil())
		}

		funcBlock = append(stmt, jen.Return(ret...))
	}

	genMethod.Jen = jen.Params(args...).Params(returns...).Block(funcBlock...)

	return nil
}

func (g *generator) buildNoLookup(
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	if err := g.getOverlappingStructDefinition(ctx, source, target); err != nil {
		return nil, nil, err
	}

	for _, rule := range BuildSteps {
		if rule.matches(ctx, source, target) {
			return rule.build(g, ctx, sourceID, source, target, errPath)
		}
	}

	return nil, nil, typeMismatch(source, target)
}

func (g *generator) assignNoLookup(
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	if err := g.getOverlappingStructDefinition(ctx, source, target); err != nil {
		return nil, err
	}

	for _, rule := range BuildSteps {
		if rule.matches(ctx, source, target) {
			return rule.assign(g, ctx, assignTo, sourceID, source, target, errPath)
		}
	}

	return nil, typeMismatch(source, target)
}

func (g *generator) convertTo(
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	if !target.Pointer || !target.PointerInner.Struct {
		return nil, NewBuildError("target type must be a pointer struct for goverter:update signatures.")
	}
	sourcePointer := false
	if !source.Struct {
		if source.Pointer && source.PointerInner.Struct {
			sourcePointer = true
			source = source.PointerInner
		} else {
			return nil, NewBuildError("source type must be a struct or pointer struct for goverter:update signatures.")
		}
	}

	var s BuildStruct
	stmt, err := s.assign(g, ctx, assignTo, sourceID, source, target.PointerInner, errPath)
	if sourcePointer {
		stmt = []jen.Code{jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...)}
	}
	return stmt, err
}

func (g *generator) CallMethod(
	ctx *MethodContext,
	definition *MethodDefinition,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	params := []jen.Code{}
	formatErr := func(s string) *BuildError {
		return NewBuildError(fmt.Sprintf("Error using method:\n    %s%s\n\n%s", definition.ID, definition.ArgDebug("        "), s))
	}

	for _, arg := range definition.RawArgs {
		switch arg.Use {
		case ArgUseInterface:
			params = append(params, jen.Id(ThisVar))
		case ArgUseContext:
			if !g.requireContext(ctx, arg.Type) {
				return nil, nil, formatErr("Could not satisfy all required context parameters:\n" + strings.Join(AvailableContextDebug(definition.Context, ctx.AvailableContext), "\n"))
			}
			if id, ok := ctx.Context[arg.Type.String]; ok {
				params = append(params, id.Code.Clone())
			}
		case ArgUseSource:
			if !source.AssignableTo(definition.Source) && !definition.TypeParams {
				cause := fmt.Sprintf("Method source type mismatches with conversion source: %s != %s", definition.Source.String, source.String)
				return nil, nil, formatErr(cause)
			}
			params = append(params, sourceID.Code)
		case ArgUseMultiSource:
			panic("multi source aren't supported right now. https://github.com/jmattheis/goverter/issues/143")
		case ArgUseTarget:
			panic("unreachable")
		}
	}

	if !definition.Target.AssignableTo(target) && !definition.TypeParams {
		cause := fmt.Sprintf("Method return type mismatches with target: %s != %s", definition.Target.String, target.String)
		return nil, nil, formatErr(cause)
	}

	qual := g.qualMethod(definition)
	if definition.ReturnError {
		name := ctx.Name(target.ID())
		ctx.SetErrorTargetVar(jen.Id(name))

		ret, ok := g.ReturnError(ctx, errPath, jen.Id("err"))
		if !ok {
			return nil, nil, formatErr("Used method returns error but conversion method does not")
		}

		stmt := []jen.Code{
			jen.List(jen.Id(name), jen.Id("err")).Op(":=").Add(qual.Call(params...)),
			jen.If(jen.Id("err").Op("!=").Nil()).Block(ret),
		}
		return stmt, VariableID(jen.Id(name)), nil
	}
	id := OtherID(qual.Call(params...))
	return nil, id, nil
}

func (g *generator) ReturnError(ctx *MethodContext, errPath ErrorPath, id *jen.Statement) (jen.Code, bool) {
	current := g.lookup.ByID(ctx.IndexID)
	if !ctx.Conf.ReturnError {
		for _, path := range append([]MethodIndexID{ctx.IndexID}, current.OriginPath...) {
			check := g.lookup.ByID(path)
			if check.Explicit && !check.ReturnError {
				return nil, false
			}

			if !check.ReturnError {
				check.ReturnError = true
				check.Dirty = true
			}
		}
	}
	returns := []jen.Code{}
	if !current.UpdateTarget {
		returns = append(returns, ctx.TargetVar)
	}
	returns = append(returns, g.wrap(ctx, errPath, id))
	return jen.Return(returns...), true
}

func (g *generator) requireContext(ctx *MethodContext, need *Type) bool {
	if _, ok := ctx.Context[need.String]; ok {
		return true
	}

	current := g.lookup.ByID(ctx.IndexID)
	for _, path := range append([]MethodIndexID{ctx.IndexID}, current.OriginPath...) {
		check := g.lookup.ByID(path)

		if _, ok := check.Context[need.String]; ok {
			continue
		}

		if check.Explicit {
			return false
		}

		check.Context[need.String] = need
		check.RawArgs = append(check.RawArgs, Arg{
			Name: "",
			Use:  ArgUseContext,
			Type: need,
		})
		check.Dirty = true
	}
	return true
}

func (g *generator) delegateMethod(
	ctx *MethodContext,
	delegateTo *MethodDefinition,
	sourceID *JenID,
) (*jen.Statement, *BuildError) {
	params := []jen.Code{}

	for _, arg := range delegateTo.RawArgs {
		switch arg.Use {
		case ArgUseInterface:
			params = append(params, jen.Id(ThisVar))
		case ArgUseContext:
			params = append(params, ctx.Context[arg.Type.String].Code.Clone())
		case ArgUseSource:
			params = append(params, sourceID.Code)
		case ArgUseMultiSource:
			panic("not supported atm")
		case ArgUseTarget:
			panic("unreachable")
		}
	}

	current := g.lookup.ByID(ctx.IndexID)

	returns := []jen.Code{g.qualMethod(delegateTo).Call(params...)}

	if delegateTo.ReturnError {
		if !current.ReturnError {
			return nil, NewBuildError(fmt.Sprintf("ReturnTypeMismatch: Cannot use\n\n    %s\n\nin\n\n    %s\n\nbecause no error is returned as second return parameter", delegateTo.OriginID, current.ID))
		}
	} else {
		if current.ReturnError {
			returns = append(returns, jen.Nil())
		}
	}
	return jen.Return(returns...), nil
}

// wrap invokes the error wrapper if feature is enabled.
func (g *generator) wrap(ctx *MethodContext, errPath ErrorPath, errStmt *jen.Statement) *jen.Statement {
	switch {
	case ctx.Conf.WrapErrorsUsing != "":
		return errPath.WrapErrorsUsing(ctx.Conf.WrapErrorsUsing, errStmt)
	case ctx.Conf.WrapErrors:
		return errPath.WrapErrors(errStmt)
	default:
		return errStmt
	}
}

// Build builds an implementation for the given source and target type, or uses an existing method for it.
func (g *generator) Build(
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	stmt, nextID, err := g.callExisting(ctx, sourceID, source, target, errPath)
	if nextID != nil || err != nil {
		return stmt, nextID, err
	}

	if g.shouldCreateSubMethod(ctx, source, target) {
		return g.createSubMethod(ctx, sourceID, source, target, errPath)
	}

	return g.buildNoLookup(ctx, sourceID, source, target, errPath)
}

// Assign builds an implementation for the given source and target type, or uses an existing method for it.
func (g *generator) Assign(
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	if assignTo.Must {
		return ToAssignable(assignTo)(g.Build(ctx, sourceID, source, target, errPath))
	}

	stmt, nextID, err := g.callExisting(ctx, sourceID, source, target, errPath)
	if nextID != nil || err != nil {
		return ToAssignable(assignTo)(stmt, nextID, err)
	}

	if g.shouldCreateSubMethod(ctx, source, target) {
		return ToAssignable(assignTo)(g.createSubMethod(ctx, sourceID, source, target, errPath))
	}

	return g.assignNoLookup(ctx, assignTo, sourceID, source, target, errPath)
}

func (g *generator) callExisting(
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	signature := signatureOf(source, target)
	if def, err := g.extend.Get(signature, ctx.AvailableContext); def != nil {
		return g.CallMethod(ctx, def, sourceID, source, target, errPath)
	} else if err != nil {
		return nil, nil, NewBuildError(err.Error())
	}
	if genMethod, err := g.lookup.Get(signature, ctx.AvailableContext); genMethod != nil {
		return g.CallMethod(ctx, genMethod.MethodDefinition, sourceID, source, target, errPath)
	} else if err != nil {
		return nil, nil, NewBuildError(err.Error())
	}
	return nil, nil, nil
}

func (g *generator) shouldCreateSubMethod(ctx *MethodContext, source, target *Type) bool {
	isCurrentPointerStructMethod := false
	if source.Struct && target.Struct {
		// This checks if we are currently inside the generation of one of the following combinations.
		// *Source -> Target
		//  Source -> *Target
		// *Source -> *Target
		isCurrentPointerStructMethod = ctx.Signature.Source == source.AsPointerType().String() ||
			ctx.Signature.Target == target.AsPointerType().String()
	}

	createSubMethod := false

	if ctx.HasSeen(source) {
		g.lookup.ByID(ctx.IndexID).Dirty = true
		createSubMethod = true
	} else if !isCurrentPointerStructMethod {
		switch {
		case source.Named && !source.Basic:
			createSubMethod = true
		case target.Named && !target.Basic:
			createSubMethod = true
		case source.Pointer && source.PointerInner.Named && !source.PointerInner.Basic:
			createSubMethod = true
		case source.Enum(&ctx.Conf.Enum).OK && target.Enum(&ctx.Conf.Enum).OK:
			createSubMethod = true
		}
		if ctx.Conf.SkipCopySameType && source.String == target.String {
			createSubMethod = false
		}
	}
	ctx.MarkSeen(source)

	return createSubMethod
}

func (g *generator) createSubMethod(
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPAth ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	name := g.namer.Name(source.UnescapedID() + "To" + strings.Title(target.UnescapedID()))
	orig := g.lookup.ByID(ctx.IndexID)

	var args []Arg
	args = append(args, Arg{
		Name: "source",
		Type: source,
		Use:  ArgUseSource,
	})

	path := append([]MethodIndexID{ctx.IndexID}, orig.OriginPath...)
	genMethod := &generatedMethod{
		OriginPath: path,
		Method: &Method{
			Common:      g.conf.Common,
			Fields:      map[string]*FieldMapping{},
			EnumMapping: &EnumMapping{Map: map[string]string{}},
			MethodDefinition: &MethodDefinition{
				OriginID:  ctx.Conf.OriginID,
				ID:        name,
				Package:   g.conf.OutputPackagePath,
				Name:      name,
				Generated: true,
				Parameters: Parameters{
					Source:    source,
					RawArgs:   args,
					Context:   map[string]*Type{},
					Signature: signatureOf(source, target),
					Target:    target,
				},
			},
		},
	}

	genMethod.IndexID, _ = g.lookup.Register(genMethod, genMethod.MethodDefinition)

	if err := g.buildMethod(genMethod, ctx.AvailableContext); err != nil {
		return nil, nil, err
	}
	return g.CallMethod(ctx, genMethod.MethodDefinition, sourceID, source, target, errPAth)
}

func (g *generator) hasMethod(ctx *MethodContext, source, target types.Type) bool {
	signature := Signature{Source: source.String(), Target: target.String()}
	return g.extend.Has(signature) || g.lookup.Has(signature)
}

func (g *generator) getOverlappingStructDefinition(ctx *MethodContext, source, target *Type) *BuildError {
	if !source.Struct || !target.Struct {
		return nil
	}

	overlapping := []Signature{
		{Source: source.AsPointerType().String(), Target: target.String},
		{Source: source.AsPointerType().String(), Target: target.AsPointerType().String()},
		{Source: source.String, Target: target.AsPointerType().String()},
	}

	for _, sig := range overlapping {
		if ctx.Signature == sig {
			continue
		}
		if def, _ := g.lookup.Get(sig, ctx.AvailableContext); def != nil && len(def.RawFieldSettings) > 0 {
			var toMethod string
			if def, _ := g.lookup.Get(ctx.Signature, ctx.AvailableContext); def != nil && def.Explicit {
				toMethod = fmt.Sprintf("to the %q method.", def.Name)
			} else {
				toMethod = fmt.Sprintf("to a newly created method with this signature:\n    func(%s) %s", source.String, target.String)
			}

			return NewBuildError(fmt.Sprintf(`Overlapping struct settings found.

Move these field related settings:
    goverter:%s

from the %q method %s

Goverter won't use %q inside the current conversion method
and therefore the defined field settings would be ignored.`, strings.Join(def.RawFieldSettings, "\n    goverter:"), def.Name, toMethod, def.Name))
		}
	}
	return nil
}

func typeMismatch(source, target *Type) *BuildError {
	if source.Pointer && !target.Pointer {
		return NewBuildError(fmt.Sprintf(`TypeMismatch: Cannot convert %s to %s
It is unclear how nil should be handled in the pointer to non pointer conversion.

You can enable useZeroValueOnPointerInconsistency to instruct goverter to use the zero value if source is nil
https://goverter.jmattheis.de/reference/useZeroValueOnPointerInconsistency

or you can define a custom conversion method with extend:
https://goverter.jmattheis.de/reference/extend`, source.T, target.T))
	}

	return NewBuildError(fmt.Sprintf(`TypeMismatch: Cannot convert %s to %s

You can define a custom conversion method with extend:
https://goverter.jmattheis.de/reference/extend`, source.T, target.T))
}

func (g *generator) qualMethod(m *MethodDefinition) *jen.Statement {
	switch {
	case m.CustomCall != nil:
		return m.CustomCall.Clone()
	case g.conf.OutputFormat == OutputFormatStruct && m.Generated:
		return jen.Id(ThisVar).Dot(m.Name)
	case g.conf.OutputFormat == OutputFormatFunction && m.Generated:
		return jen.Id(m.Name)
	default:
		return jen.Qual(m.Package, m.Name)
	}
}

func signatureOf(source, target *Type) Signature {
	return Signature{Source: source.String, Target: target.String}
}
