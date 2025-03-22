package main

import (
	"os"
	"strings"

	"github.com/jmattheis/goverter"
	"github.com/jmattheis/goverter/cli"
)

func main() {
	opts := cli.RunOpts{
		EnumTransformers: map[string]goverter.EnumTransformer{
			"trim-prefix": trimPrefix,
		},
	}
	cli.Run(os.Args, opts)
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
