package config

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/jmattheis/goverter"
)

type Converter struct {
	goverter.ConverterConfig
	Package  string
	FileName string
	typ      types.Type
	Methods  []*Method

	Location string
}

func (c *Converter) typeForMethod() types.Type {
	if c.OutputFormat == goverter.OutputFormatFunction {
		return nil
	}
	return c.typ
}

func (c *Converter) requireStruct() error {
	if c.OutputFormat == goverter.OutputFormatStruct {
		return nil
	}
	return fmt.Errorf("not allowed when using goverter:variables")
}

func (c *Converter) IDString() string {
	if c.typ == nil {
		return "var definition"
	}
	return c.typ.String()
}

func defaultOutputFile(name string) string {
	f := filepath.Base(name)
	ext := filepath.Ext(f)
	return strings.TrimSuffix(f, ext) + ".gen" + ext
}

func parseConverter(ctx *CfgContext, rawConverter *goverter.RawConverter, global goverter.RawLines) (*Converter, error) {
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

	resolveOutputPackage(ctx, c)

	err = parseMethods(ctx, rawConverter, c)
	return c, err
}

func resolveOutputPackage(ctx *CfgContext, c *Converter) {
	targetPackage, err := resolvePackage(c.FileName, c.Package, c.OutputFile)
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

func initConverter(loader *goverter.PackageLoader, rawConverter *goverter.RawConverter) (*Converter, error) {
	c := &Converter{
		FileName: rawConverter.FileName,
		Package:  rawConverter.PackagePath,
		Location: rawConverter.Converter.Location,
	}

	if rawConverter.InterfaceName != "" {
		c.ConverterConfig = goverter.DefaultConfigInterface
		_, interfaceObj, err := loader.GetOneRaw(c.Package, rawConverter.InterfaceName)
		if err != nil {
			return nil, err
		}

		c.typ = interfaceObj.Type()
		c.Name = rawConverter.InterfaceName + "Impl"
		return c, nil
	}

	c.ConverterConfig = goverter.DefaultConfigVariables
	c.OutputFile = defaultOutputFile(rawConverter.FileName)
	c.OutputPackageName = rawConverter.PackageName
	c.OutputPackagePath = rawConverter.PackagePath
	return c, nil
}

func parseConverterLines(ctx *CfgContext, c *Converter, source string, raw goverter.RawLines) error {
	for _, value := range raw.Lines {
		if err := parseConverterLine(ctx, c, value); err != nil {
			return formatLineError(raw, source, value, err)
		}
	}

	return nil
}

func parseConverterLine(ctx *CfgContext, c *Converter, value string) (err error) {
	cmd, rest := goverter.ParseCommand(value)
	switch cmd {
	case "converter", "variables":
		// only a marker interface
	case "name":
		if err = c.requireStruct(); err != nil {
			return err
		}
		c.Name, err = goverter.ParseString(rest)
	case "output:raw":
		c.OutputRaw = append(c.OutputRaw, rest)
	case goverter.ConfigOutputFile:
		c.OutputFile, err = goverter.ParseFile(ctx.WorkDir, rest)
	case "output:format":
		if len(c.Extend) != 0 {
			return fmt.Errorf("Cannot change output:format after extend functions have been added.\nMove the extend below the output:format setting.")
		}

		c.OutputFormat, err = goverter.ParseEnum(false, rest, goverter.OutputFormatFunction, goverter.OutputFormatStruct, goverter.OutputFormatVariable)
		if err != nil {
			return err
		}

		if c.typ == nil && c.OutputFormat != goverter.OutputFormatVariable {
			return fmt.Errorf("unsupported format for goverter:variables")
		}
		if c.typ != nil && c.OutputFormat == goverter.OutputFormatVariable {
			return fmt.Errorf("unsupported format for goverter:converter")
		}
	case "output:package":
		c.OutputPackageName = ""
		var pkg string
		pkg, err = goverter.ParseString(rest)

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
		var pattern goverter.EnumIDPattern
		pattern, err = parseIDPattern(c.Package, rest)
		c.Enum.Excludes = append(c.Enum.Excludes, pattern)
	case goverter.ConfigExtend:
		for _, name := range strings.Fields(rest) {
			opts := &goverter.ParseMethodOpts{
				ErrorPrefix:       "error parsing type",
				OutputPackagePath: c.OutputPackagePath,
				Converter:         c.typeForMethod(),
				Params:            goverter.ParamsRequired,
				ContextMatch:      c.ArgContextRegex,
			}
			var defs []*goverter.MethodDefinition
			defs, err = ctx.Loader.GetMatching(c.Package, name, opts)
			if err != nil {
				break
			}
			c.Extend = append(c.Extend, defs...)
		}
	default:
		_, err = parseCommon(&c.Common, cmd, rest)
	}
	return err
}
