package generator

import (
	"github.com/dave/jennifer/jen"
	"github.com/jmattheis/goverter/config"
	builder2 "github.com/jmattheis/goverter/generator/internal/builder"
)

// Config the generate config.
type Config struct {
	BuildConstraint string
}

// BuildSteps that'll used for generation.
var BuildSteps = []builder2.Builder{
	&builder2.UseUnderlyingTypeMethods{},
	&builder2.SkipCopy{},
	&builder2.Enum{},
	&builder2.BasicTargetPointerRule{},
	&builder2.Pointer{},
	&builder2.SourcePointer{},
	&builder2.TargetPointer{},
	&builder2.Basic{},
	&builder2.Struct{},
	&builder2.List{},
	&builder2.Map{},
}

// Generate generates a jen.File containing converters.
func Generate(converters []*config.Converter, c Config) (map[string][]byte, error) {
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

func generateConverter(converter *config.Converter, f *jen.File, n *builder2.Namer) error {
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
