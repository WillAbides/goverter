package goverter

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	EnumActionPanic  = "@panic"
	EnumActionError  = "@error"
	EnumActionIgnore = "@ignore"
)

type ConfiguredTransformer struct {
	Name        string
	Transformer EnumTransformer
	Config      string
}

type EnumMapping struct {
	Transformers []ConfiguredTransformer
	Map          map[string]string
}

var DefaultEnumTransformers = map[string]EnumTransformer{
	"regex": func(ctx TransformEnumContext) (map[string]string, error) {
		parts := strings.Split(ctx.Config, " ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config, expected two strings separated by space")
		}

		pattern, err := regexp.Compile(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", parts[0], err)
		}

		m := map[string]string{}
		for key := range ctx.Source.Members {
			targetKey := pattern.ReplaceAllString(key, parts[1])
			if _, ok := ctx.Target.Members[targetKey]; ok {
				m[key] = targetKey
			}
		}
		return m, nil
	},
}

func IsEnumAction(s string) bool {
	return strings.HasPrefix(s, "@")
}

func ValidateEnumAction(s string) error {
	switch s {
	case EnumActionPanic, EnumActionError, EnumActionIgnore:
		return nil
	default:
		return fmt.Errorf("invalid enum action %q, must be one of %q, %q, or %q", s, EnumActionPanic, EnumActionIgnore, EnumActionError)
	}
}

func ParseTransformer(ctx *cfgContext, name, config string) (ConfiguredTransformer, error) {
	t, ok := ctx.EnumTransformers[name]
	if !ok {
		t, ok = DefaultEnumTransformers[name]
	}

	if !ok {
		return ConfiguredTransformer{}, fmt.Errorf("transformer %q does not exist", name)
	}

	return ConfiguredTransformer{Name: name, Transformer: t, Config: config}, nil
}

func ParseIDPattern(cwd, rest string) (pattern EnumIDPattern, err error) {
	path, name, err := parseMethodString(cwd, rest)
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
