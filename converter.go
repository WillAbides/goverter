package goverter

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"
)

type OutputFormat string

const (
	OutputFormatStruct   OutputFormat = "struct"
	OutputFormatVariable OutputFormat = "assign-variable"
	OutputFormatFunction OutputFormat = "function"
)

var DefaultConfigInterface = ConverterConfig{
	OutputFile:   "./generated/generated.go",
	Common:       DefaultCommon,
	OutputFormat: OutputFormatStruct,
}

var DefaultConfigVariables = ConverterConfig{
	OutputFormat: OutputFormatVariable,
	Common:       DefaultCommon,
}

var DefaultCommon = Common{
	Enum: EnumConfig{Enabled: true},
}

type ConverterConfig struct {
	Common
	Name              string
	OutputRaw         []string
	OutputFile        string
	OutputPackagePath string
	OutputPackageName string
	OutputFormat      OutputFormat
	Extend            []*MethodDefinition
	Comments          []string
}

func (conf *ConverterConfig) PackageID() string {
	if conf.OutputPackageName == "" {
		return conf.OutputPackagePath
	}
	return conf.OutputPackagePath + ":" + conf.OutputPackageName
}

const (
	ConfigExtend     = "extend"
	ConfigOutputFile = "output:file"
)

type Converter struct {
	ConverterConfig
	Package  string
	FileName string
	Type     types.Type
	Methods  []*Method

	Location string
}

func (c *Converter) TypeForMethod() types.Type {
	if c.OutputFormat == OutputFormatFunction {
		return nil
	}
	return c.Type
}

func (c *Converter) RequireStruct() error {
	if c.OutputFormat == OutputFormatStruct {
		return nil
	}
	return fmt.Errorf("not allowed when using goverter:variables")
}

func (c *Converter) IDString() string {
	if c.Type == nil {
		return "var definition"
	}
	return c.Type.String()
}

func DefaultOutputFile(name string) string {
	f := filepath.Base(name)
	ext := filepath.Ext(f)
	return strings.TrimSuffix(f, ext) + ".gen" + ext
}

func ParseConverter(
	ctx *CfgContext,
	rawConverter *RawConverter,
	global RawLines,
) (*Converter, error) {
	c, err := InitConverter(ctx.Loader, rawConverter)
	if err != nil {
		return nil, err
	}

	if err := ParseConverterLines(ctx, c, "global", global); err != nil {
		return nil, err
	}
	if err := ParseConverterLines(ctx, c, c.IDString(), rawConverter.Converter); err != nil {
		return nil, err
	}

	ResolveOutputPackage(ctx, c)

	err = ParseMethodsCfg(ctx, rawConverter, c)
	return c, err
}

func ResolveOutputPackage(ctx *CfgContext, c *Converter) {
	targetPackage, err := ResolvePackage(c.FileName, c.Package, c.OutputFile)
	if err != nil {
		return
	}

	if c.OutputPackagePath == "" {
		c.OutputPackagePath = targetPackage
	}

	pkg := ctx.Loader.GetUncheckedPkg(targetPackage)

	if pkg == nil {
		return
	}

	if c.OutputPackageName == "" {
		c.OutputPackageName = pkg.Types.Name()
	}
}

func InitConverter(loader *PackageLoader, rawConverter *RawConverter) (*Converter, error) {
	c := &Converter{
		FileName: rawConverter.FileName,
		Package:  rawConverter.PackagePath,
		Location: rawConverter.Converter.Location,
	}

	if rawConverter.InterfaceName != "" {
		c.ConverterConfig = DefaultConfigInterface
		_, interfaceObj, err := loader.GetOneRaw(c.Package, rawConverter.InterfaceName)
		if err != nil {
			return nil, err
		}

		c.Type = interfaceObj.Type()
		c.Name = rawConverter.InterfaceName + "Impl"
		return c, nil
	}

	c.ConverterConfig = DefaultConfigVariables
	c.OutputFile = DefaultOutputFile(rawConverter.FileName)
	c.OutputPackageName = rawConverter.PackageName
	c.OutputPackagePath = rawConverter.PackagePath
	return c, nil
}

func ParseConverterLines(ctx *CfgContext, c *Converter, source string, raw RawLines) error {
	for _, value := range raw.Lines {
		if err := ParseConverterLine(ctx, c, value); err != nil {
			return FormatLineError(raw, source, value, err)
		}
	}

	return nil
}

func ParseConverterLine(ctx *CfgContext, c *Converter, value string) (err error) {
	cmd, rest := ParseCommand(value)
	switch cmd {
	case "converter", "variables":
		// only a marker interface
	case "name":
		if err = c.RequireStruct(); err != nil {
			return err
		}
		c.Name, err = ParseString(rest)
	case "output:raw":
		c.OutputRaw = append(c.OutputRaw, rest)
	case ConfigOutputFile:
		c.OutputFile, err = ParseFile(ctx.WorkDir, rest)
	case "output:format":
		if len(c.Extend) != 0 {
			return fmt.Errorf("Cannot change output:format after extend functions have been added.\nMove the extend below the output:format setting.")
		}

		c.OutputFormat, err = ParseEnum(false, rest, OutputFormatFunction, OutputFormatStruct, OutputFormatVariable)
		if err != nil {
			return err
		}

		if c.Type == nil && c.OutputFormat != OutputFormatVariable {
			return fmt.Errorf("unsupported format for goverter:variables")
		}
		if c.Type != nil && c.OutputFormat == OutputFormatVariable {
			return fmt.Errorf("unsupported format for goverter:converter")
		}
	case "output:package":
		c.OutputPackageName = ""
		var pkg string
		pkg, err = ParseString(rest)

		parts := strings.SplitN(pkg, ":", 2)
		switch len(parts) {
		case 2:
			c.OutputPackageName = parts[1]
			fallthrough
		case 1:
			c.OutputPackagePath = parts[0]
		}
	case "struct:comment":
		if err = c.RequireStruct(); err != nil {
			return err
		}
		c.Comments = append(c.Comments, rest)
	case "enum:exclude":
		var pattern EnumIDPattern
		pattern, err = ParseIDPattern(c.Package, rest)
		c.Enum.Excludes = append(c.Enum.Excludes, pattern)
	case ConfigExtend:
		for _, name := range strings.Fields(rest) {
			opts := &ParseMethodOpts{
				ErrorPrefix:       "error parsing type",
				OutputPackagePath: c.OutputPackagePath,
				Converter:         c.TypeForMethod(),
				Params:            ParamsRequired,
				ContextMatch:      c.ArgContextRegex,
			}
			var defs []*MethodDefinition
			defs, err = ctx.Loader.GetMatching(c.Package, name, opts)
			if err != nil {
				break
			}
			c.Extend = append(c.Extend, defs...)
		}
	default:
		_, err = ParseCommon(&c.Common, cmd, rest)
	}
	return err
}
