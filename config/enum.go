package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jmattheis/goverter"
)

func parseTransformer(ctx *CfgContext, name, config string) (goverter.ConfiguredTransformer, error) {
	t, ok := ctx.EnumTransformers[name]
	if !ok {
		t, ok = goverter.DefaultEnumTransformers[name]
	}

	if !ok {
		return goverter.ConfiguredTransformer{}, fmt.Errorf("transformer %q does not exist", name)
	}

	return goverter.ConfiguredTransformer{Name: name, Transformer: t, Config: config}, nil
}

func IsEnumAction(s string) bool {
	return strings.HasPrefix(s, "@")
}

func validateEnumAction(s string) error {
	switch s {
	case goverter.EnumActionPanic, goverter.EnumActionError, goverter.EnumActionIgnore:
		return nil
	default:
		return fmt.Errorf("invalid enum action %q, must be one of %q, %q, or %q", s, goverter.EnumActionPanic, goverter.EnumActionIgnore, goverter.EnumActionError)
	}
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
