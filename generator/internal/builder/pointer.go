package builder

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

// Pointer handles pointer types.
type Pointer struct{}

// Matches returns true, if the builder can create handle the given types.
func (*Pointer) Matches(_ *goverter.MethodContext, source, target *goverter.Type) bool {
	return source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (p *Pointer) Build(gen goverter.Generator, ctx *goverter.MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := goverter.BuildTargetVar(gen, ctx, sourceID, source, target, errPath)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, goverter.AssignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
		if err != nil {
			return nil, nil, err.Lift(&goverter.ErrorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, goverter.VariableID(valueVar), nil
	}

	return goverter.BuildByAssign(p, gen, ctx, sourceID, source, target, errPath)
}

func (*Pointer) Assign(gen goverter.Generator, ctx *goverter.MethodContext, assignTo *goverter.AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, errPath goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	nextBlock, id, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target.PointerInner, errPath)
	if err != nil {
		return nil, err.Lift(&goverter.ErrorMessagePath{
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
func (*SourcePointer) Matches(ctx *goverter.MethodContext, source, target *goverter.Type) bool {
	return ctx.Conf.UseZeroValueOnPointerInconsistency && source.Pointer && !target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (s *SourcePointer) Build(gen goverter.Generator, ctx *goverter.MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	if ctx.UseConstructor && ctx.Conf.DefaultUpdate {
		buildStmt, valueVar, err := goverter.BuildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, goverter.AssignOf(valueVar).IsUpdate(), sourceID.Deref(source), source.PointerInner, target, path)
		if err != nil {
			return nil, nil, err.Lift(&goverter.ErrorMessagePath{
				SourceID:   "*",
				SourceType: source.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, jen.If(sourceID.Code.Clone().Op("!=").Nil()).Block(stmt...))

		return buildStmt, goverter.VariableID(valueVar), nil
	}

	return goverter.BuildByAssign(s, gen, ctx, sourceID, source, target, path)
}

func (*SourcePointer) Assign(gen goverter.Generator, ctx *goverter.MethodContext, assignTo *goverter.AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	nextInner, nextID, err := gen.Build(ctx, sourceID.Deref(source), source.PointerInner, target, path)
	if err != nil {
		return nil, err.Lift(&goverter.ErrorMessagePath{
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
func (*TargetPointer) Matches(_ *goverter.MethodContext, source, target *goverter.Type) bool {
	return !source.Pointer && target.Pointer
}

// Build creates conversion source code for the given source and target type.
func (*TargetPointer) Build(gen goverter.Generator, ctx *goverter.MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	ctx.SetErrorTargetVar(jen.Nil())

	if ctx.UseConstructor {
		buildStmt, valueVar, err := goverter.BuildTargetVar(gen, ctx, sourceID, source, target, path)
		if err != nil {
			return nil, nil, err
		}

		stmt, err := gen.Assign(ctx, goverter.AssignOf(jen.Parens(jen.Op("*").Add(valueVar))).IsUpdate(), sourceID, source, target.PointerInner, path)
		if err != nil {
			return nil, nil, err.Lift(&goverter.ErrorMessagePath{
				TargetID:   "*",
				TargetType: target.PointerInner.String,
			})
		}

		buildStmt = append(buildStmt, stmt...)

		return buildStmt, goverter.VariableID(valueVar), nil
	}

	stmt, id, err := gen.Build(ctx, sourceID, source, target.PointerInner, path)
	if err != nil {
		return nil, nil, err.Lift(&goverter.ErrorMessagePath{
			TargetID:   "*",
			TargetType: target.PointerInner.String,
		})
	}

	pstmt, nextID := id.Pointer(target.PointerInner, ctx.Name)
	stmt = append(stmt, pstmt...)

	return stmt, nextID, nil
}

func (tp *TargetPointer) Assign(gen goverter.Generator, ctx *goverter.MethodContext, assignTo *goverter.AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return goverter.AssignByBuild(tp, gen, ctx, assignTo, sourceID, source, target, path)
}
