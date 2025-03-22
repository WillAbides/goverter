package goverter

import (
	"github.com/dave/jennifer/jen"
)

// GenerateConfig the generate config.
type GenerateConfig struct {
	BuildConstraint string
}

// buildSteps that'll be used for generation.
func buildSteps() []builder {
	return []builder{
		&UseUnderlyingTypeMethods{},
		&SkipCopy{},
		&BuildEnum{},
		&BasicTargetPointerRule{},
		&Pointer{},
		&SourcePointer{},
		&TargetPointer{},
		&basic{},
		&BuildStruct{},
		&BuildList{},
		&BuildMap{},
	}
}

// Generate generates a jen.File containing converters.
func Generate(converters []*Converter, c GenerateConfig) (map[string][]byte, error) {
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

func generateConverter(converter *Converter, f *jen.File, n *Namer) error {
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

func setupGenerator(converter *Converter, n *Namer) (*generator, error) {
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
