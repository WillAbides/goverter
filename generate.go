package goverter

import (
	"github.com/dave/jennifer/jen"
)

// generateConfig the generate config.
type generateConfig struct {
	BuildConstraint string
}

// buildSteps that'll be used for generation.
func buildSteps() []builder {
	return []builder{
		&useUnderlyingTypeMethods{},
		&skipCopy{},
		&buildEnum{},
		&basicTargetPointerRule{},
		&pointer{},
		&sourcePointer{},
		&targetPointer{},
		&basic{},
		&buildStruct{},
		&buildList{},
		&buildMap{},
	}
}

// generateFiles generates files and returns them as a map of file paths to file contents.
func generateFiles(converters []*converter, c generateConfig) (map[string][]byte, error) {
	manager := &fileManager{Files: map[string]*managedFile{}}

	for _, converter := range converters {
		jenFile, n, err := manager.Get(converter, c)
		if err != nil {
			return nil, err
		}

		if err := generateConverter(converter, jenFile, n); err != nil {
			return nil, err
		}
	}

	return manager.renderFiles()
}

func generateConverter(converter *converter, f *jen.File, n *namer) error {
	gen, err := setupGenerator(converter, n)
	if err != nil {
		return err
	}

	if err := validateMethods(gen.lookup); err != nil {
		return err
	}

	if err := gen.buildMethods(f); err != nil {
		return err
	}
	return nil
}

func setupGenerator(converter *converter, n *namer) (*generator, error) {
	extend := newMethodIndex[methodDefinition]()
	for _, def := range converter.Extend {
		extend.RegisterOverrideOverlapping(def, def)
	}

	var err error
	lookup := newMethodIndex[generatedMethod]()
	for _, cMethod := range converter.Methods {
		gen := &generatedMethod{
			method:   cMethod,
			Dirty:    true,
			Explicit: true,
		}
		if gen.UpdateTarget {
			gen.IndexID, err = lookup.RegisterUpdate(gen)
		} else {
			gen.IndexID, err = lookup.Register(gen, gen.methodDefinition)
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
