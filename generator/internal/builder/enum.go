package builder

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
)

type Enum struct{}

// Matches returns true, if the builder can create handle the given types.
func (*Enum) Matches(ctx *MethodContext, source, target *goverter.Type) bool {
	return isEnum(ctx, source, target)
}

func isEnum(ctx *MethodContext, source, target *goverter.Type) bool {
	return ctx.Conf.Enum.Enabled &&
		source.Enum(&ctx.Conf.Enum).OK &&
		target.Enum(&ctx.Conf.Enum).OK
}

// Build creates conversion source code for the given source and target type.
func (*Enum) Build(gen Generator, ctx *MethodContext, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.JenID, *goverter.BuildError) {
	stmt, nameVar, err := buildTargetVar(gen, ctx, sourceID, source, target, path)
	if err != nil {
		return nil, nil, err
	}

	var cases []jen.Code

	targetEnum := target.Enum(&ctx.Conf.Enum)
	sourceEnum := source.Enum(&ctx.Conf.Enum)

	definedKeys := ctx.DefinedEnumFields(target)

	transformerMapping, err := executeTransformers(ctx.Conf.EnumMapping.Transformers, source, target, sourceEnum, targetEnum)
	if err != nil {
		return nil, nil, err
	}

	sourceTargetMapping := map[interface{}]enumMapping{}
	for _, sourceName := range sourceEnum.SortedMembers() {
		delete(definedKeys, sourceName)

		targetName, ok := ctx.Conf.EnumMapping.Map[sourceName]
		if !ok {
			targetName, ok = transformerMapping[sourceName]
		}

		if !ok {
			targetName = sourceName
		}

		sourceQual := jen.Qual(source.NamedType.Obj().Pkg().Path(), sourceName)
		body, err := caseAction(gen, ctx, nameVar, target, targetEnum, targetName, sourceID, path)
		if err != nil {
			return nil, nil, err.Lift(&goverter.ErrorMessagePath{
				SourceType: fmtEnumValue(sourceEnum, sourceName),
				SourceID:   sourceName,
				Prefix:     ".",
				TargetID:   targetName,
				TargetType: "???",
			})
		}

		sourceValue := sourceEnum.Members[sourceName]
		if previous, ok := sourceTargetMapping[sourceValue]; ok {
			if enumTargetMismatches(previous, targetEnum, targetName) {
				return nil, nil, enumTargetMismatchError(targetEnum, sourceName, targetName, previous, sourceValue).Lift(&goverter.ErrorMessagePath{
					SourceType: fmtEnumValue(sourceEnum, sourceName),
					SourceID:   sourceName,
					Prefix:     ".",
					TargetID:   targetName,
					TargetType: fmtEnumValue(targetEnum, targetName),
				})
			} else {
				cases = append(cases, jen.Comment(fmt.Sprintf("Skipped %s -> %s because it duplicates %s -> %s",
					fmtEnumValue(sourceEnum, sourceName), fmtEnumValue(targetEnum, targetName),
					fmtEnumValue(sourceEnum, previous.Source), fmtEnumValue(targetEnum, previous.Target))))
			}
		} else {
			sourceTargetMapping[sourceValue] = enumMapping{Source: sourceName, Target: targetName}
			cases = append(cases, jen.Case(sourceQual).Add(body))
		}
	}

	enumUnknown := ctx.Conf.Common.Enum.Unknown
	if enumUnknown == "" {
		return nil, nil, goverter.NewBuildError("Enum detected but enum:unknown is not configured.\nSee https://goverter.jmattheis.de/guide/enum")
	}

	body, err := caseAction(gen, ctx, nameVar, target, targetEnum, enumUnknown, sourceID, path)
	if err != nil {
		return nil, nil, err.Lift(&goverter.ErrorMessagePath{
			SourceID:   "@enum:unknown",
			Prefix:     ".",
			TargetID:   enumUnknown,
			TargetType: "???",
		})
	}
	cases = append(cases, jen.Default().Add(body))

	for name := range definedKeys {
		return nil, nil, goverter.NewBuildError(fmt.Sprintf("Configured enum value %s does not exist on\n    %s", name, source.String)).
			Lift(&goverter.ErrorMessagePath{
				Prefix:     ".",
				SourceID:   name,
				SourceType: "???",
			})
	}

	stmt = append(stmt, jen.Switch(sourceID.Code).Block(cases...))
	return stmt, goverter.VariableID(nameVar), nil
}

func (s *Enum) Assign(gen Generator, ctx *MethodContext, assignTo *AssignTo, sourceID *goverter.JenID, source, target *goverter.Type, path goverter.ErrorPath) ([]jen.Code, *goverter.BuildError) {
	return AssignByBuild(s, gen, ctx, assignTo, sourceID, source, target, path)
}

func caseAction(gen Generator, ctx *MethodContext, nameVar *jen.Statement, target *goverter.Type, targetEnum *goverter.Enum, targetName string, sourceID *goverter.JenID, errPath goverter.ErrorPath) (jen.Code, *goverter.BuildError) {
	if goverter.IsEnumAction(targetName) {
		switch targetName {
		case goverter.EnumActionIgnore:
			return jen.Comment("ignored"), nil
		case goverter.EnumActionPanic:
			return jen.Panic(jen.Qual("fmt", "Sprintf").Call(jen.Lit("unexpected enum element: %v"), sourceID.Code.Clone())), nil
		case goverter.EnumActionError:
			errStmt := jen.Qual("fmt", "Errorf").Call(jen.Lit("unexpected enum element: %v"), sourceID.Code.Clone())
			code, ok := gen.ReturnError(ctx, errPath, errStmt)
			if !ok {
				return nil, goverter.NewBuildError(fmt.Sprintf("Cannot return %s because the explicitly defined conversion method doesn't return an error.", goverter.EnumActionError))
			}
			return code, nil
		default:
			return nil, goverter.NewBuildError(fmt.Sprintf("invalid target %q", targetName))
		}
	}
	_, ok := targetEnum.Members[targetName]
	if !ok {
		return nil, goverter.NewBuildError(fmt.Sprintf("Enum %s does not exist on\n    %s\n\nSee https://goverter.jmattheis.de/guide/enum", targetName, target.String))
	}

	targetQual := jen.Qual(target.NamedType.Obj().Pkg().Path(), targetName)
	return nameVar.Clone().Op("=").Add(targetQual), nil
}

func executeTransformers(transformers []goverter.ConfiguredTransformer, source, target *goverter.Type, sourceEnum, targetEnum *goverter.Enum) (map[string]string, *goverter.BuildError) {
	transformerMapping := map[string]string{}
	for _, t := range transformers {
		m, err := t.Transformer(goverter.TransformEnumContext{
			Source: goverter.Enum{OK: true, Type: source.NamedType, Members: sourceEnum.Members},
			Target: goverter.Enum{OK: true, Type: target.NamedType, Members: targetEnum.Members},
			Config: t.Config,
		})
		if err != nil {
			return nil, goverter.NewBuildError(fmt.Sprintf("error executing transformer %q with config %q: %s", t.Name, t.Config, err))
		}
		if len(m) == 0 {
			return nil, goverter.NewBuildError(fmt.Sprintf("transformer %q with config %q did not return any mapped values. Is there an configuration error?", t.Name, t.Config))
		}
		for key, value := range m {
			transformerMapping[key] = value
		}
	}
	return transformerMapping, nil
}

func enumTargetMismatches(previous enumMapping, targetEnum *goverter.Enum, targetName string) bool {
	if !goverter.IsEnumAction(targetName) && !goverter.IsEnumAction(previous.Target) {
		return targetEnum.Members[previous.Target] != targetEnum.Members[targetName]
	}
	return targetName != previous.Target
}

func enumTargetMismatchError(targetEnum *goverter.Enum, sourceName, targetName string, previous enumMapping, sourceValue interface{}) *goverter.BuildError {
	return goverter.NewBuildError(fmt.Sprintf(`Detected multiple enum source members with the same value but different target values/actions.
    %s(%v) -> %s
    %s(%v) -> %s

Explicitly define what the correct mapping is. E.g. by adding
    goverter:enum:map %s %s
    goverter:enum:map %s %s

See https://goverter.jmattheis.de/guide/enum#mapping-enum-keys`,
		previous.Source, sourceValue, fmtEnumValue(targetEnum, previous.Target),
		sourceName, sourceValue, fmtEnumValue(targetEnum, targetName),
		previous.Source, previous.Target,
		sourceName, previous.Target))
}

func fmtEnumValue(targetEnum *goverter.Enum, targetName string) string {
	if goverter.IsEnumAction(targetName) {
		return fmt.Sprintf("%s(action)", targetName)
	}
	return fmt.Sprintf("%s(%v)", targetName, targetEnum.Members[targetName])
}

type enumMapping struct {
	Target string
	Source string
}
