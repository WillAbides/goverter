package config

import (
	"fmt"
	"regexp"

	"github.com/jmattheis/goverter"
)

func parseTransformer(ctx *goverter.CfgContext, name, config string) (goverter.ConfiguredTransformer, error) {
	t, ok := ctx.EnumTransformers[name]
	if !ok {
		t, ok = goverter.DefaultEnumTransformers[name]
	}

	if !ok {
		return goverter.ConfiguredTransformer{}, fmt.Errorf("transformer %q does not exist", name)
	}

	return goverter.ConfiguredTransformer{Name: name, Transformer: t, Config: config}, nil
}

func parseIDPattern(cwd, rest string) (pattern goverter.EnumIDPattern, err error) {
	path, name, err := goverter.ParseMethodString(cwd, rest)
	if err != nil {
		return pattern, err
	}

	pattern.Path, err = regexp.Compile(path)
	if err != nil {
		return pattern, err
	}
	pattern.Name, err = regexp.Compile(name)
	if err != nil {
		return pattern, err
	}
	return pattern, nil
}
