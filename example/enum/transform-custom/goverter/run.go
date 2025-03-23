package main

import (
	"os"
	"strings"

	"github.com/jmattheis/goverter"
)

func main() {
	opts := goverter.RunOpts{
		EnumTransformers: map[string]goverter.EnumTransformer{
			"trim-prefix": trimPrefix,
		},
	}
	goverter.Run(os.Args, opts)
}

func trimPrefix(ctx goverter.TransformEnumContext) (map[string]string, error) {
	m := map[string]string{}
	for key := range ctx.Source.Members {
		targetKey := strings.TrimPrefix(key, ctx.Config)
		if _, ok := ctx.Target.Members[targetKey]; ok {
			m[key] = targetKey
		}
	}
	return m, nil
}
