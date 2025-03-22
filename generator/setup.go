package generator

import (
	"github.com/jmattheis/goverter"
)

func setupGenerator(converter *goverter.Converter, n *goverter.Namer) (*generator, error) {
	extend := goverter.NewMethodIndex[goverter.MethodDefinition]()
	for _, def := range converter.Extend {
		extend.RegisterOverrideOverlapping(def, def)
	}

	var err error
	lookup := goverter.NewMethodIndex[generatedMethod]()
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
