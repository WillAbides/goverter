package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jmattheis/goverter/enum"
)

const (
	EnumActionPanic  = "@panic"
	EnumActionError  = "@error"
	EnumActionIgnore = "@ignore"
)

type EnumMapping struct {
	Transformers []ConfiguredTransformer
	Map          map[string]string
}

type ConfiguredTransformer struct {
	Name        string
	Transformer EnumTransformer
	Config      string
}

func parseTransformer(ctx *context, name, config string) (ConfiguredTransformer, error) {
	t, ok := ctx.EnumTransformers[name]
	if !ok {
		t, ok = defaultEnumTransformers[name]
	}

	if !ok {
		return ConfiguredTransformer{}, fmt.Errorf("transformer %q does not exist", name)
	}

	return ConfiguredTransformer{Name: name, Transformer: t, Config: config}, nil
}

func IsEnumAction(s string) bool {
	return strings.HasPrefix(s, "@")
}

func validateEnumAction(s string) error {
	switch s {
	case EnumActionPanic, EnumActionError, EnumActionIgnore:
		return nil
	default:
		return fmt.Errorf("invalid enum action %q, must be one of %q, %q, or %q", s, EnumActionPanic, EnumActionIgnore, EnumActionError)
	}
}

func parseIDPattern(cwd, rest string) (pattern enum.EnumIDPattern, err error) {
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

var defaultEnumTransformers = map[string]EnumTransformer{
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

// EnumTransformer transforms a source enum members to target enum members
//
// The transformer must only return keys present inside the
// context.Source.Members and context.Target.Members, if something cannot be
// mapped by the transformer just skip the key and don't return it. An error by
// this methods aborts the aborts the whole goverter conversion, so only use it
// when there are config errors.
type EnumTransformer func(context TransformEnumContext) (map[string]string, error)

type TransformEnumContext struct {
	Source enum.Enum
	Target enum.Enum
	// Config is user definable config
	Config string
}
