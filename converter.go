package goverter

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"
)

type outputFormat string

const (
	OutputFormatStruct   outputFormat = "struct"
	OutputFormatVariable outputFormat = "assign-variable"
	OutputFormatFunction outputFormat = "function"
)

var defaultConfigInterface = converterConfig{
	OutputFile:   "./generated/generated.go",
	commonCfg:    defaultCommon,
	OutputFormat: OutputFormatStruct,
}

var defaultConfigVariables = converterConfig{
	OutputFormat: OutputFormatVariable,
	commonCfg:    defaultCommon,
}

var defaultCommon = commonCfg{
	Enum: enumConfig{enabled: true},
}

type converterConfig struct {
	commonCfg
	Name              string
	OutputRaw         []string
	OutputFile        string
	OutputPackagePath string
	OutputPackageName string
	OutputFormat      outputFormat
	Extend            []*methodDefinition
	Comments          []string
}

func (conf *converterConfig) PackageID() string {
	if conf.OutputPackageName == "" {
		return conf.OutputPackagePath
	}
	return conf.OutputPackagePath + ":" + conf.OutputPackageName
}

const (
	configExtend     = "extend"
	configOutputFile = "output:file"
)

type Converter struct {
	converterConfig
	Package  string
	FileName string
	Type     types.Type
	Methods  []*method

	Location string
}

func (c *Converter) typeForMethod() types.Type {
	if c.OutputFormat == OutputFormatFunction {
		return nil
	}
	return c.Type
}

func (c *Converter) requireStruct() error {
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

func defaultOutputFile(name string) string {
	f := filepath.Base(name)
	ext := filepath.Ext(f)
	return strings.TrimSuffix(f, ext) + ".gen" + ext
}

func parseConverter(ctx *cfgContext, rawConverter *RawConverter, global RawLines) (*Converter, error) {
	c, err := initConverter(ctx.Loader, rawConverter)
	if err != nil {
		return nil, err
	}

	if err := parseConverterLines(ctx, c, "global", global); err != nil {
		return nil, err
	}
	if err := parseConverterLines(ctx, c, c.IDString(), rawConverter.Converter); err != nil {
		return nil, err
	}

	ResolveOutputPackage(ctx, c)

	err = parseMethodsCfg(ctx, rawConverter, c)
	return c, err
}

func ResolveOutputPackage(ctx *cfgContext, c *Converter) {
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

func initConverter(loader *PackageLoader, rawConverter *RawConverter) (*Converter, error) {
	c := &Converter{
		FileName: rawConverter.FileName,
		Package:  rawConverter.PackagePath,
		Location: rawConverter.Converter.Location,
	}

	if rawConverter.InterfaceName != "" {
		c.converterConfig = defaultConfigInterface
		_, interfaceObj, err := loader.GetOneRaw(c.Package, rawConverter.InterfaceName)
		if err != nil {
			return nil, err
		}

		c.Type = interfaceObj.Type()
		c.Name = rawConverter.InterfaceName + "Impl"
		return c, nil
	}

	c.converterConfig = defaultConfigVariables
	c.OutputFile = defaultOutputFile(rawConverter.FileName)
	c.OutputPackageName = rawConverter.PackageName
	c.OutputPackagePath = rawConverter.PackagePath
	return c, nil
}

func parseConverterLines(ctx *cfgContext, c *Converter, source string, raw RawLines) error {
	for _, value := range raw.Lines {
		if err := parseConverterLine(ctx, c, value); err != nil {
			return formatLineError(raw, source, value, err)
		}
	}

	return nil
}

func parseConverterLine(ctx *cfgContext, c *Converter, value string) (err error) {
	cmd, rest := parseCommand(value)
	switch cmd {
	case "converter", "variables":
		// only a marker interface
	case "name":
		if err = c.requireStruct(); err != nil {
			return err
		}
		c.Name, err = parseString(rest)
	case "output:raw":
		c.OutputRaw = append(c.OutputRaw, rest)
	case configOutputFile:
		c.OutputFile, err = parseFile(ctx.WorkDir, rest)
	case "output:format":
		if len(c.Extend) != 0 {
			return fmt.Errorf("Cannot change output:format after extend functions have been added.\nMove the extend below the output:format setting.")
		}

		c.OutputFormat, err = parseEnum(false, rest, OutputFormatFunction, OutputFormatStruct, OutputFormatVariable)
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
		pkg, err = parseString(rest)

		parts := strings.SplitN(pkg, ":", 2)
		switch len(parts) {
		case 2:
			c.OutputPackageName = parts[1]
			fallthrough
		case 1:
			c.OutputPackagePath = parts[0]
		}
	case "struct:comment":
		if err = c.requireStruct(); err != nil {
			return err
		}
		c.Comments = append(c.Comments, rest)
	case "enum:exclude":
		var pattern EnumIDPattern
		pattern, err = ParseIDPattern(c.Package, rest)
		c.Enum.excludes = append(c.Enum.excludes, pattern)
	case configExtend:
		for _, name := range strings.Fields(rest) {
			opts := &parseMethodOpts{
				ErrorPrefix:       "error parsing type",
				OutputPackagePath: c.OutputPackagePath,
				Converter:         c.typeForMethod(),
				Params:            paramsRequired,
				ContextMatch:      c.ArgContextRegex,
			}
			var defs []*methodDefinition
			defs, err = ctx.Loader.GetMatching(c.Package, name, opts)
			if err != nil {
				break
			}
			c.Extend = append(c.Extend, defs...)
		}
	default:
		_, err = parseCommon(&c.commonCfg, cmd, rest)
	}
	return err
}
