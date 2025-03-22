package cli

import (
	"os"
	"path/filepath"

	"github.com/jmattheis/goverter"
	"github.com/jmattheis/goverter/generator"
)

// GenerateConfig the config for generating a converter.
type GenerateConfig struct {
	// PackagePatterns are golang package patterns to scan, required.
	PackagePatterns []string
	// WorkingDir is the working directory (usually the location of go.mod file), can be empty.
	WorkingDir string
	// Global are the global config commands that will be applied to all converters
	Global goverter.RawLines
	// BuildTags is a comma separated list passed to -tags when scanning for conversion interfaces.
	BuildTags string
	// OutputBuildConstraint will be added as go:build constraints to all files.
	OutputBuildConstraint string
	// EnumTransformers describes additional enum transformers usable in the enum:transform setting.
	EnumTransformers map[string]goverter.EnumTransformer
}

// GenerateConverters generates converters.
func GenerateConverters(c *GenerateConfig) error {
	files, err := generateConvertersRaw(c)
	if err != nil {
		return err
	}

	return writeFiles(files)
}

func generateConvertersRaw(c *GenerateConfig) (map[string][]byte, error) {
	rawConverters, err := goverter.ParseDocs(goverter.ParseDocsConfig{
		BuildTags:      c.BuildTags,
		PackagePattern: c.PackagePatterns,
		WorkingDir:     c.WorkingDir,
	})
	if err != nil {
		return nil, err
	}

	converters, err := goverter.ParseRaw(&goverter.Raw{
		BuildTags:  c.BuildTags,
		WorkDir:    c.WorkingDir,
		Converters: rawConverters,
		Global:     c.Global,

		OuputBuildConstraint: c.OutputBuildConstraint,

		EnumTransformers: c.EnumTransformers,
	})
	if err != nil {
		return nil, err
	}

	return generator.Generate(converters, generator.Config{
		BuildConstraint: c.OutputBuildConstraint,
	})
}

func writeFiles(files map[string][]byte) error {
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}
