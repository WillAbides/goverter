package generator

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter"
	"github.com/jmattheis/goverter/generator/internal/builder"
)

// Config the generate config.
type Config struct {
	BuildConstraint string
}

// BuildSteps that'll used for generation.
var BuildSteps = []goverter.Builder{
	&builder.UseUnderlyingTypeMethods{},
	&builder.SkipCopy{},
	&goverter.BuildEnum{},
	&goverter.BasicTargetPointerRule{},
	&builder.Pointer{},
	&builder.SourcePointer{},
	&builder.TargetPointer{},
	&goverter.Basic{},
	&builder.Struct{},
	&goverter.BuildList{},
	&goverter.BuildMap{},
}

// Generate generates a jen.File containing converters.
func Generate(converters []*goverter.Converter, c Config) (map[string][]byte, error) {
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

func generateConverter(converter *goverter.Converter, f *jen.File, n *goverter.Namer) error {
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
