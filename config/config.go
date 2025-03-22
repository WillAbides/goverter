package config

import (
	"fmt"
	"sort"

	"github.com/jmattheis/goverter"
)

type CfgContext struct {
	Loader           *goverter.PackageLoader
	WorkDir          string
	EnumTransformers map[string]goverter.EnumTransformer
}

func ParseRaw(raw *goverter.Raw) ([]*Converter, error) {
	loader, err := goverter.NewPackageLoader(raw.WorkDir, raw.BuildTags, getPackages(raw))
	if err != nil {
		return nil, err
	}

	ctx := &CfgContext{Loader: loader, EnumTransformers: raw.EnumTransformers, WorkDir: raw.WorkDir}

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

func formatLineError(lines goverter.RawLines, t, value string, err error) error {
	cmd, _ := goverter.ParseCommand(value)
	msg := `error parsing 'goverter:%s' at
    %s
    %s

%s`
	return fmt.Errorf(msg, cmd, lines.Location, t, err)
}
