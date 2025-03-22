package goverter

import (
	"github.com/dave/jennifer/jen"
)

// Pointer handles pointer types.
type Pointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*Pointer) matches(_ *MethodContext, source, target *Type) bool {
	return source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (p *Pointer) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, errPath)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, AssignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
		if err != nil {
			return nil, nil, err.Lift(&ErrorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, VariableID(valueVar), nil
	}

	return buildByAssign(p, gen, ctx, sourceID, source, target, errPath)
}

func (*Pointer) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	nextBlock, id, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
	if err != nil {
		return nil, err.Lift(&ErrorMessagePath{
			SourceID:   "*",
			SourceType: source.PointerInner.String,
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}

	pstmt, tmpID := id.Pointer(target.PointerInner, ctx.Name)

	ifBlock := append(nextBlock, pstmt...)
	ifBlock = append(ifBlock, assignTo.Stmt.Clone().Op("=").Add(tmpID.Code))

	stmt := []jen.Code{
		jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(ifBlock...),
	}

	return stmt, err
}

// SourcePointer handles type were only the source is a pointer.
type SourcePointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*SourcePointer) matches(ctx *MethodContext, source, target *Type) bool {
	return ctx.Conf.UseZeroValueOnPointerInconsistency && source.Pointer && !target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (s *SourcePointer) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	path ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, AssignOf(valueVar).IsUpdate(), sourceID.Deref(source), source.PointerInner, target, path)
		if err != nil {
			return nil, nil, err.Lift(&ErrorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, VariableID(valueVar), nil
	}

	return buildByAssign(s, gen, ctx, sourceID, source, target, path)
}

func (*SourcePointer) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	path ErrorPath,
) ([]jen.Code, *BuildError) {
	nextInner, nextID, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target, path)
	if err != nil {
		return nil, err.Lift(&ErrorMessagePath{
			SourceID:   "*",
			SourceType: source.PointerInner.String,
		})
	}

	stmt := []jen.Code{
		jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(
			append(nextInner, assignTo.Stmt.Clone().Op("=").Add(nextID.Code))...,
		),
	}

	return stmt, nil
}

// TargetPointer handles type were only the target is a pointer.
type TargetPointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*TargetPointer) matches(_ *MethodContext, source, target *Type) bool {
	return !source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (*TargetPointer) build(
	gen *generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	path ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	if ctx.UseConstructor {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, AssignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID, source, target.PointerInner, path)
		if err != nil {
			return nil, nil, err.Lift(&ErrorMessagePath{
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, stmt...)

		return buildStmt, VariableID(valueVar), nil
	}

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, path)
	if err != nil {
		return nil, nil, err.Lift(&ErrorMessagePath{
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}

	pstmt, nextID := id.Pointer(target.PointerInner, ctx.Name)
	stmt = append(stmt, pstmt...)

	return stmt, nextID, nil
}

func (tp *TargetPointer) assign(
	gen *generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	path ErrorPath,
) ([]jen.Code, *BuildError) {
	return assignByBuild(tp, gen, ctx, assignTo, sourceID, source, target, path)
}
