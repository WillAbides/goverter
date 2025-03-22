package main

import (
	"os"
	"strings"

	"github.com/jmattheis/goverter/cli"
	"github.com/jmattheis/goverter/config"
)

func main() {
	opts := cli.RunOpts{
		EnumTransformers: map[string]config.EnumTransformer{
			"trim-prefix": trimPrefix,
		},
	}
	cli.Run(os.Args, opts)
}

func trimPrefix(ctx config.TransformEnumContext) (map[string]string, error) {
	m := map[string]string{}
	for key := range ctx.Source.Members {
		targetKey := strings.TrimPrefix(key, ctx.Config)
		if _, ok := ctx.Target.Members[targetKey]; ok {
			m[key] = targetKey
		}
	}
	return m, nil
}
