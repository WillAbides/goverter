package goverter

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
)

type RunOpts struct {
	EnumTransformers map[string]EnumTransformer
}

// GenerateCmdConfig the config for generating a converter.
type GenerateCmdConfig struct {
	// PackagePatterns are golang package patterns to scan, required.
	PackagePatterns []string
	// WorkingDir is the working directory (usually the location of go.mod file), can be empty.
	WorkingDir string
	// Global are the global config commands that will be applied to all converters
	Global RawLines
	// BuildTags is a comma separated list passed to -tags when scanning for conversion interfaces.
	BuildTags string
	// OutputBuildConstraint will be added as go:build constraints to all files.
	OutputBuildConstraint string
	// EnumTransformers describes additional enum transformers usable in the enum:transform setting.
	EnumTransformers map[string]EnumTransformer
}

// GenerateConverters generates converters.
func GenerateConverters(c *GenerateCmdConfig) error {
	files, err := GenerateConvertersRaw(c)
	if err != nil {
		return err
	}

	return WriteFiles(files)
}

func GenerateConvertersRaw(c *GenerateCmdConfig) (map[string][]byte, error) {
	rawConverters, err := ParseDocs(ParseDocsConfig{
		BuildTags:      c.BuildTags,
		PackagePattern: c.PackagePatterns,
		WorkingDir:     c.WorkingDir,
	})
	if err != nil {
		return nil, err
	}

	converters, err := ParseRaw(&Raw{
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

	return Generate(converters, GenerateConfig{
		BuildConstraint: c.OutputBuildConstraint,
	})
}

func WriteFiles(files map[string][]byte) error {
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

type Command interface {
	_c()
}

type GenerateCmd struct {
	Config *GenerateCmdConfig
}

type HelpCmd struct {
	Usage string
}

type VersionCmd struct{}

func (*HelpCmd) _c()     {}
func (*GenerateCmd) _c() {}
func (*VersionCmd) _c()  {}

type stringVals []string

func (s stringVals) String() string {
	return fmt.Sprint([]string(s))
}

func (s *stringVals) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return nil, usageErr("invalid args", "unknown")
	}
	cmd := args[0]

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &HelpCmd{Usage: usage(cmd)}, nil
		}
		return nil, usageErr(err.Error(), cmd)
	}

	subArgs := fs.Args()
	if len(subArgs) == 0 {
		return nil, usageErr("missing command", cmd)
	}

	switch subArgs[0] {
	case "gen":
		return parseGen(cmd, subArgs[1:])
	case "version":
		return &VersionCmd{}, nil
	case "help":
		return &HelpCmd{Usage: usage(cmd)}, nil
	default:
		return nil, usageErr("unknown command "+subArgs[0], cmd)
	}
}

func parseGen(cmd string, args []string) (Command, error) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	var global stringVals
	fs.Var(&global, "global", "")
	fs.Var(&global, "g", "")

	buildTags := fs.String("build-tags", "goverter", "")
	outputConstraint := fs.String("output-constraint", "!goverter", "")
	cwd := fs.String("cwd", "", "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &HelpCmd{Usage: usage(cmd)}, nil
		}
		return nil, usageErr(err.Error(), cmd)
	}

	patterns := fs.Args()

	if len(patterns) == 0 {
		return nil, usageErr("missing PATTERN", cmd)
	}

	c := GenerateCmdConfig{
		PackagePatterns:       patterns,
		BuildTags:             *buildTags,
		OutputBuildConstraint: *outputConstraint,
		WorkingDir:            *cwd,
		EnumTransformers:      map[string]EnumTransformer{},
		Global: RawLines{
			Lines:    global,
			Location: "command line (-g, -global)",
		},
	}
	return &GenerateCmd{Config: &c}, nil
}

func usageErr(err, cmd string) error {
	return fmt.Errorf("Error: %s\n%s", err, usage(cmd))
}

func usage(cmd string) string {
	return fmt.Sprintf(`Usage:
  %s gen [OPTIONS] PACKAGE...
  %s help
  %s version

PACKAGE(s):
  Define the import paths goverter will use to search for converter interfaces.
  You can define multiple packages and use the special ... golang pattern to
  select multiple packages. See $ go help packages

OPTIONS:
  -build-tags [tags]: (default: goverter)
      a comma-separated list of additional build tags to consider satisfied
      during the loading of conversion interfaces. See 'go help buildconstraint'.
      Can be disabled by supplying an empty string.

  -cwd [value]:
      set the working directory

  -g [value], -global [value]:
      apply settings to all defined converters. For a list of available
      settings see: https://goverter.jmattheis.de/reference/settings

  -output-constraint [constraint]: (default: !goverter)
      A build constraint added to all files generated by goverter.
      Can be disabled by supplying an empty string.

Examples:
  %s gen ./example/simple ./example/complex
  %s gen ./example/...
  %s gen github.com/jmattheis/goverter/example/simple
  %s gen -g 'ignoreMissing no' -g 'skipCopySameType' ./simple

Documentation:
  Full documentation is available here: https://goverter.jmattheis.de`, cmd, cmd, cmd, cmd, cmd, cmd, cmd)
}

// Run runs the goverter cli with the given args and customizations.
func Run(args []string, opts RunOpts) {
	cmd, err := Parse(args)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch cmd := cmd.(type) {
	case *HelpCmd:
		_, _ = fmt.Fprintln(os.Stdout, cmd.Usage)
		os.Exit(0)
	case *GenerateCmd:
		if opts.EnumTransformers != nil {
			for key, value := range opts.EnumTransformers {
				cmd.Config.EnumTransformers[key] = value
			}
		}

		if err = GenerateConverters(cmd.Config); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *VersionCmd:
		b, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Println(b)
		}
	default:
		panic("unknown command")
	}
}
