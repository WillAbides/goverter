package goverter

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type RawLines struct {
	Location string
	Lines    []string
}

// hasSetting returns true if r.Lines contains a line for the given setting.
func (r RawLines) hasSetting(setting string) bool {
	return slices.ContainsFunc(r.Lines, func(line string) bool {
		if !strings.HasPrefix(line, setting) {
			return false
		}
		line = strings.TrimPrefix(line, setting)
		if line == "" {
			return true
		}
		return line[0] == ':' || line[0] == ' '
	})
}

type RawConverter struct {
	PackagePath   string
	PackageName   string
	InterfaceName string
	Converter     RawLines
	Methods       map[string]RawLines
	FileName      string
}

// EnumTransformer transforms a source enum members to target enum members
//
// The transformer must only return keys present inside the
// context.Source.Members and context.Target.Members, if something cannot be
// mapped by the transformer just skip the key and don't return it. An error by
// this methods aborts the whole goverter conversion, so only use it
// when there are config errors.
type EnumTransformer func(context TransformEnumContext) (map[string]string, error)

type TransformEnumContext struct {
	Source enum
	Target enum
	// Config is user definable config
	Config string
}

type Raw struct {
	Converters []RawConverter
	Global     RawLines

	WorkDir              string
	BuildTags            string
	OuputBuildConstraint string

	EnumTransformers map[string]EnumTransformer
}

func formatLineError(lines RawLines, t, value string, err error) error {
	cmd, _ := parseCommand(value)
	msg := `error parsing 'goverter:%s' at
    %s
    %s

%s`
	return fmt.Errorf(msg, cmd, lines.Location, t, err)
}

type cfgContext struct {
	Loader           *PackageLoader
	WorkDir          string
	EnumTransformers map[string]EnumTransformer
}

func ParseRaw(raw *Raw) ([]*Converter, error) {
	loader, err := NewPackageLoader(raw.WorkDir, raw.BuildTags, getPackages(raw))
	if err != nil {
		return nil, err
	}

	ctx := &cfgContext{Loader: loader, EnumTransformers: raw.EnumTransformers, WorkDir: raw.WorkDir}

	var converters []*Converter
	for _, rawConverter := range raw.Converters {
		converter, err := parseConverter(ctx, &rawConverter, raw.Global)
		if err != nil {
			return nil, err
		}
		converters = append(converters, converter)
	}

	sort.Slice(converters, func(i, j int) bool {
		return converters[i].Name < converters[j].Name
	})

	return converters, nil
}
