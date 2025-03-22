package config

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

// HasSetting returns true if r.Lines contains a line for the given setting.
func (r RawLines) HasSetting(setting string) bool {
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

type Raw struct {
	Converters []RawConverter
	Global     RawLines

	WorkDir              string
	BuildTags            string
	OuputBuildConstraint string

	EnumTransformers map[string]EnumTransformer
}

type context struct {
	Loader           *packageLoader
	WorkDir          string
	EnumTransformers map[string]EnumTransformer
}

func Parse(raw *Raw) ([]*Converter, error) {
	loader, err := newPackageLoader(raw.WorkDir, raw.BuildTags, getPackages(raw))
	if err != nil {
		return nil, err
	}

	ctx := &context{Loader: loader, EnumTransformers: raw.EnumTransformers, WorkDir: raw.WorkDir}

	converters := []*Converter{}
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

func formatLineError(lines RawLines, t, value string, err error) error {
	cmd, _ := parseCommand(value)
	msg := `error parsing 'goverter:%s' at
    %s
    %s

%s`
	return fmt.Errorf(msg, cmd, lines.Location, t, err)
}
