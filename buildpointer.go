package goverter

import (
	"github.com/dave/jennifer/jen"
)

// pointer handles pointer types.
type pointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*pointer) matches(_ *methodContext, source, target *xType) bool {
	return source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (p *pointer) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *jenID, *buildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, errPath)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, assignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
		if err != nil {
			return nil, nil, err.Lift(&errorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, variableID(valueVar), nil
	}

	return buildByAssign(p, gen, ctx, sourceID, source, target, errPath)
}

func (*pointer) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	errPath errorPath,
) ([]jen.Code, *buildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	nextBlock, id, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
	if err != nil {
		return nil, err.Lift(&errorMessagePath{
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

// sourcePointer handles type were only the source is a pointer.
type sourcePointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*sourcePointer) matches(ctx *methodContext, source, target *xType) bool {
	return ctx.Conf.UseZeroValueOnPointerInconsistency && source.Pointer && !target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (s *sourcePointer) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *jenID, *buildError) {
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, assignOf(valueVar).IsUpdate(), sourceID.Deref(source), source.PointerInner, target, path)
		if err != nil {
			return nil, nil, err.Lift(&errorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, variableID(valueVar), nil
	}

	return buildByAssign(s, gen, ctx, sourceID, source, target, path)
}

func (*sourcePointer) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *buildError) {
	nextInner, nextID, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target, path)
	if err != nil {
		return nil, err.Lift(&errorMessagePath{
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

// targetPointer handles type were only the target is a pointer.
type targetPointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*targetPointer) matches(_ *methodContext, source, target *xType) bool {
	return !source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (*targetPointer) build(
	gen *generator,
	ctx *methodContext,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *jenID, *buildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	if ctx.UseConstructor {
		buildStmt, valueVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, assignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID, source, target.PointerInner, path)
		if err != nil {
			return nil, nil, err.Lift(&errorMessagePath{
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, stmt...)

		return buildStmt, variableID(valueVar), nil
	}

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, path)
	if err != nil {
		return nil, nil, err.Lift(&errorMessagePath{
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}

	pstmt, nextID := id.Pointer(target.PointerInner, ctx.Name)
	stmt = append(stmt, pstmt...)

	return stmt, nextID, nil
}

func (tp *targetPointer) assign(
	gen *generator,
	ctx *methodContext,
	assignTo *assignTo,
	sourceID *jenID,
	source, target *xType,
	path errorPath,
) ([]jen.Code, *buildError) {
	return assignByBuild(tp, gen, ctx, assignTo, sourceID, source, target, path)
}
