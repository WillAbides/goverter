package config

import (
	"sort"

	"github.com/jmattheis/goverter"
)

func ParseRaw(raw *goverter.Raw) ([]*Converter, error) {
	loader, err := goverter.NewPackageLoader(raw.WorkDir, raw.BuildTags, goverter.GetPackages(raw))
	if err != nil {
		return nil, err
	}

	ctx := &goverter.CfgContext{Loader: loader, EnumTransformers: raw.EnumTransformers, WorkDir: raw.WorkDir}

	var converters []*Converter
	for _, rawConverter := range raw.Converters {
		converter, err := ParseConverter(ctx, &rawConverter, raw.Global)
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
