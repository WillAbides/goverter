package generator

import (
	"github.com/jmattheis/goverter/config"
	"github.com/jmattheis/goverter/generator/internal/builder"
	"github.com/jmattheis/goverter/xtype/method"
)

func setupGenerator(converter *config.Converter, n *builder.Namer) (*generator, error) {
	extend := method.NewIndex[method.MethodDefinition]()
	for _, def := range converter.Extend {
		extend.RegisterOverrideOverlapping(def, def)
	}

	var err error
	lookup := method.NewIndex[generatedMethod]()
	for _, cMethod := range converter.Methods {
		gen := &generatedMethod{
			Method:   cMethod,
			Dirty:    true,
			Explicit: true,
		}
		if gen.UpdateTarget {
			gen.IndexID, err = lookup.RegisterUpdate(gen)
		} else {
			gen.IndexID, err = lookup.Register(gen, gen.MethodDefinition)
		}
		if err != nil {
			return nil, err
		}
	}

	gen := generator{
		namer:  n,
		conf:   converter,
		lookup: lookup,
		extend: extend,
	}

	return &gen, nil
}
