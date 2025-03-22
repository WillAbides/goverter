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
