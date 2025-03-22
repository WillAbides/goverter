package goverter

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/dave/jennifer/jen"
)

// BuildStruct handles struct types.
type BuildStruct struct{}

// Matches returns true, if the builder can create handle the given types.
func (*BuildStruct) matches(_ *MethodContext, source, target *Type) bool {
	return source.Struct && target.Struct
}

// Build creates conversion source code for the given source and target type.
func (s *BuildStruct) build(
	gen Generator,
	ctx *MethodContext,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *JenID, *BuildError) {
	// Optimization for golang sets
	if !source.Named && !target.Named && source.StructType.NumFields() == 0 && target.StructType.NumFields() == 0 {
		return nil, sourceID, nil
	}
	return buildByAssign(s, gen, ctx, sourceID, source, target, errPath)
}

func (s *BuildStruct) assign(
	gen Generator,
	ctx *MethodContext,
	assignTo *AssignTo,
	sourceID *JenID,
	source, target *Type,
	errPath ErrorPath,
) ([]jen.Code, *BuildError) {
	additionalFieldSources, err := parseAutoMap(ctx, source)
	if err != nil {
		return nil, err
	}

	var stmt []jen.Code

	definedFields := ctx.DefinedFields(target)
	usedSourceID := false
	for i := 0; i < target.StructType.NumFields(); i++ {
		targetField := target.StructType.Field(i)
		delete(definedFields, targetField.Name())

		fieldMapping := ctx.Field(target, targetField.Name())

		if fieldMapping.Ignore {
			continue
		}
		if !targetField.Exported() && ctx.Conf.IgnoreUnexported {
			continue
		}

		if !Accessible(targetField, ctx.OutputPackagePath) {
			cause := unexportedStructError(targetField.Name(), source.String, target.String)
			return nil, NewBuildError(cause).Lift(&ErrorMessagePath{
				Prefix:     ".",
				SourceID:   "???",
				TargetID:   targetField.Name(),
				TargetType: targetField.Type().String(),
			})
		}

		targetFieldType := TypeOf(targetField.Type())
		targetFieldPath := errPath.Field(targetField.Name())

		if fieldMapping.Function == nil {
			usedSourceID = true
			nextID, nextSource, mapStmt, lift, skip, err := mapField(gen, ctx, targetField, sourceID, source, target, additionalFieldSources, targetFieldPath)
			if skip {
				continue
			}
			if err != nil {
				return nil, err
			}
			stmt = append(stmt, mapStmt...)

			fieldStmt, err := gen.Assign(ctx, AssignOf(assignTo.Stmt.Clone().Dot(targetField.Name())), nextID, nextSource, targetFieldType, targetFieldPath)
			if err != nil {
				return nil, err.Lift(lift...)
			}
			if shouldCheckAgainstZero(ctx, nextSource, targetFieldType, assignTo.Update, false) {
				stmt = append(stmt, jen.If(nextID.Code.Clone().Op("!=").Add(zeroValue(nextSource.T))).Block(fieldStmt...))
			} else {
				stmt = append(stmt, fieldStmt...)
			}
		} else {
			def := fieldMapping.Function

			sourceLift := []*ErrorMessagePath{}
			var functionCallSourceID *JenID
			var functionCallSourceType *Type
			if def.Source != nil {
				usedSourceID = true
				nextID, nextSource, mapStmt, mapLift, _, err := mapField(gen, ctx, targetField, sourceID, source, target, additionalFieldSources, targetFieldPath)
				if err != nil {
					return nil, err
				}
				sourceLift = mapLift
				stmt = append(stmt, mapStmt...)

				if fieldMapping.Source == "." && sourceID.ParentPointer != nil &&
					def.Source.AssignableTo(source.AsPointer()) {
					functionCallSourceID = sourceID.ParentPointer
					functionCallSourceType = source.AsPointer()
				} else {
					functionCallSourceID = nextID
					functionCallSourceType = nextSource
				}
			} else {
				sourceLift = append(sourceLift, &ErrorMessagePath{
					Prefix:     ".",
					TargetID:   targetField.Name(),
					TargetType: targetFieldType.String,
				})
			}

			callStmt, callReturnID, err := gen.CallMethod(ctx, fieldMapping.Function, functionCallSourceID, functionCallSourceType, targetFieldType, targetFieldPath)
			if err != nil {
				return nil, err.Lift(sourceLift...)
			}
			callStmt = append(callStmt, assignTo.Stmt.Clone().Dot(targetField.Name()).Op("=").Add(callReturnID.Code))

			if shouldCheckAgainstZero(ctx, functionCallSourceType, targetFieldType, assignTo.Update, true) {
				stmt = append(stmt, jen.If(functionCallSourceID.Code.Clone().Op("!=").Add(zeroValue(functionCallSourceType.T))).Block(callStmt...))
			} else {
				stmt = append(stmt, callStmt...)
			}
		}
	}
	if !usedSourceID {
		stmt = append(stmt, jen.Id("_").Op("=").Add(sourceID.Code.Clone()))
	}

	for name := range definedFields {
		return nil, NewBuildError(fmt.Sprintf("Field %q does not exist.\nRemove or adjust field settings referencing this field.", name)).Lift(&ErrorMessagePath{
			Prefix:     ".",
			TargetID:   name,
			TargetType: "???",
		})
	}

	return stmt, nil
}

func shouldCheckAgainstZero(ctx *MethodContext, s, t *Type, isUpdate, call bool) bool {
	switch {
	case !ctx.Conf.UpdateTarget && !isUpdate:
		return false
	case s.Struct && ctx.Conf.IgnoreStructZeroValueField:
		return true
	case s.Basic && ctx.Conf.IgnoreBasicZeroValueField:
		return true
	case ctx.Conf.IgnoreNillableZeroValueField:
		if s.Chan || s.Map || s.Func || s.Signature || s.Interface {
			return true
		}
		if call || (ctx.Conf.SkipCopySameType && types.Identical(s.T, t.T)) {
			return (s.List && !s.ListFixed) || s.Pointer
		}
		return false
	default:
		return false
	}
}

var structMethodContextRegex = regexp.MustCompile(".*")

func mapField(
	gen Generator,
	ctx *MethodContext,
	targetField *types.Var,
	sourceID *JenID,
	source, target *Type,
	additionalFieldSources []FieldSources,
	errPath ErrorPath,
) (*JenID, *Type, []jen.Code, []*ErrorMessagePath, bool, *BuildError) {
	lift := []*ErrorMessagePath{}
	def := ctx.Field(target, targetField.Name())
	pathString := def.Source
	if pathString == "." {
		lift = append(lift, &ErrorMessagePath{
			Prefix:     ".",
			SourceID:   " ",
			SourceType: "goverter:map . " + targetField.Name(),
			TargetID:   targetField.Name(),
			TargetType: targetField.Type().String(),
		})
		return sourceID, source, nil, lift, false, nil
	}

	var path []string
	if pathString == "" {
		sourceMatch, err := FindField(targetField.Name(), ctx.Conf.MatchIgnoreCase, source, additionalFieldSources)
		if err != nil {
			cause := fmt.Sprintf("Cannot match the target field with the source entry: %s.", err.Error())
			skip := false
			if ctx.Conf.IgnoreMissing {
				_, skip = err.(*NoMatchError)
			}
			return nil, nil, nil, nil, skip, NewBuildError(cause).Lift(&ErrorMessagePath{
				Prefix:     ".",
				SourceID:   "???",
				TargetID:   targetField.Name(),
				TargetType: targetField.Type().String(),
			})
		}

		path = sourceMatch.Path
	} else {
		path = strings.Split(pathString, ".")
	}

	var condition *jen.Statement

	nextIDCode := sourceID.Code
	nextSource := source

	for i := 0; i < len(path); i++ {
		if nextSource.Pointer {
			addCondition := nextIDCode.Clone().Op("!=").Nil()
			if condition == nil {
				condition = addCondition
			} else {
				condition = condition.Clone().Op("&&").Add(addCondition)
			}
			nextSource = nextSource.PointerInner
		}
		if !nextSource.Struct {
			cause := fmt.Sprintf("Cannot access '%s' on %s.", path[i], nextSource.T)
			return nil, nil, nil, nil, false, NewBuildError(cause).Lift(&ErrorMessagePath{
				Prefix:     ".",
				SourceID:   path[i],
				SourceType: "???",
			}).Lift(lift...)
		}
		sourceMatch, err := FindExactField(nextSource, path[i])
		if err == nil {
			nextSource = sourceMatch.Type
			nextIDCode = nextIDCode.Clone().Dot(sourceMatch.Name)
			liftPath := &ErrorMessagePath{
				Prefix:     ".",
				SourceID:   sourceMatch.Name,
				SourceType: nextSource.String,
			}

			if i == len(path)-1 {
				liftPath.TargetID = targetField.Name()
				liftPath.TargetType = targetField.Type().String()
			}
			lift = append(lift, liftPath)
			continue
		}

		cause := fmt.Sprintf("Cannot find the mapped field on the source entry: %s.", err.Error())
		return nil, nil, []jen.Code{}, nil, false, NewBuildError(cause).Lift(&ErrorMessagePath{
			Prefix:     ".",
			SourceID:   path[i],
			SourceType: "???",
		}).Lift(lift...)
	}

	returnID := VariableID(nextIDCode)
	var innerStmt []jen.Code
	if nextSource.Func {
		def, err := ParseMethod(nextSource.FuncType, &ParseMethodOpts{
			Converter:         nil,
			OutputPackagePath: ctx.OutputPackagePath,
			ErrorPrefix:       "Error parsing struct method",
			Params:            ParamsNone,
			ContextMatch:      structMethodContextRegex,
			CustomCall:        nextIDCode,
		}, EmptyLocalMethodOpts)
		if err != nil {
			return nil, nil, nil, nil, false, NewBuildError(err.Error()).Lift(lift...)
		}

		methodCallInner, callID, callErr := gen.CallMethod(ctx, def, nil, nil, def.Target, errPath)
		if callErr != nil {
			return nil, nil, nil, nil, false, callErr.Lift(lift...)
		}
		innerStmt = methodCallInner
		nextSource = def.Target
		returnID = callID
		lift = append(lift, &ErrorMessagePath{
			Prefix:     "(",
			SourceID:   ")",
			SourceType: def.Target.String,
		})
	}

	if condition != nil && !nextSource.Pointer {
		lift[len(lift)-1].SourceType = fmt.Sprintf("*%s (It is a pointer because the nested property in the goverter:map was a pointer)",
			lift[len(lift)-1].SourceType)
	}

	stmt := []jen.Code{}
	if condition != nil {
		pointerNext := nextSource
		if !nextSource.Pointer {
			pointerNext = nextSource.AsPointer()
		}
		tempName := ctx.Name(pointerNext.ID())
		stmt = append(stmt, jen.Var().Id(tempName).Add(pointerNext.TypeAsJen()))

		if nextSource.Pointer {
			innerStmt = append(innerStmt, jen.Id(tempName).Op("=").Add(returnID.Code))
		} else {
			pstmt, pointerID := returnID.Pointer(nextSource, ctx.Name)
			innerStmt = append(innerStmt, pstmt...)
			innerStmt = append(innerStmt, jen.Id(tempName).Op("=").Add(pointerID.Code))
		}

		stmt = append(stmt, jen.If(condition).Block(innerStmt...))
		nextSource = pointerNext
		returnID = VariableID(jen.Id(tempName))
	} else {
		stmt = append(stmt, innerStmt...)
	}

	return returnID, nextSource, stmt, lift, false, nil
}

func parseAutoMap(ctx *MethodContext, source *Type) ([]FieldSources, *BuildError) {
	fieldSources := []FieldSources{}
	for _, field := range ctx.Conf.AutoMap {
		innerSource := source
		lift := []*ErrorMessagePath{}
		path := strings.Split(field, ".")
		for _, part := range path {
			field, err := FindExactField(innerSource, part)
			if err != nil {
				return nil, NewBuildError(err.Error()).Lift(&ErrorMessagePath{
					Prefix:     ".",
					SourceID:   part,
					SourceType: "goverter:autoMap",
				}).Lift(lift...)
			}
			lift = append(lift, &ErrorMessagePath{
				Prefix:     ".",
				SourceID:   field.Name,
				SourceType: field.Type.String,
			})
			innerSource = field.Type

			switch {
			case innerSource.Pointer && innerSource.PointerInner.Struct:
				innerSource = TypeOf(innerSource.PointerInner.StructType)
			case innerSource.Struct:
				// ok
			default:
				return nil, NewBuildError(fmt.Sprintf("%s is not a struct or struct pointer", part)).Lift(lift...)
			}
		}

		fieldSources = append(fieldSources, FieldSources{Path: path, Type: innerSource})
	}
	return fieldSources, nil
}

func unexportedStructError(targetField, sourceType, targetType string) string {
	return fmt.Sprintf(`Cannot set value for unexported field "%s".

See https://goverter.jmattheis.de/guide/unexported-field`, targetField)
}

func zeroValue(t types.Type) *jen.Statement {
	switch cast := t.(type) {
	case *types.Basic:
		if cast.Info()&types.IsString != 0 {
			return jen.Lit("")
		} else if cast.Info()&types.IsNumeric != 0 {
			return jen.Lit(0)
		} else if cast.Info()&types.IsBoolean != 0 {
			return jen.Lit(false)
		}
		panic("unknown basic type" + cast.String())
	case *types.Named:
		switch under := cast.Underlying().(type) {
		case *types.Struct:
			return jen.Parens(ToCode(t).Block())
		default:
			return zeroValue(under)
		}
	case *types.Struct, *types.Array:
		return ToCode(t).Block()
	case *types.Interface, *types.Signature, *types.Pointer, *types.Map, *types.Slice, *types.Chan:
		return jen.Nil()
	}
	panic("unsupported type " + t.String())
}
