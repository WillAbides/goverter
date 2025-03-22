package goverter

import (
	"fmt"
	"go/types"
	"strings"
)

const (
	ConfigMap     = "map"
	ConfigDefault = "default"
)

type FieldMapping struct {
	Source   string
	Function *MethodDefinition
	Ignore   bool
}

type Method struct {
	*MethodDefinition
	Common

	Constructor *MethodDefinition
	AutoMap     []string
	Fields      map[string]*FieldMapping
	EnumMapping *EnumMapping

	RawFieldSettings []string

	Location    string
	UpdateParam string
	LocalOpts   LocalMethodOpts
}

func (m *Method) Field(targetName string) *FieldMapping {
	target, ok := m.Fields[targetName]
	if !ok {
		target = &FieldMapping{}
		m.Fields[targetName] = target
	}
	return target
}

func ParseMethodMap(remaining string) (source, target, custom string, err error) {
	parts := strings.SplitN(remaining, "|", 2)
	if len(parts) == 2 {
		custom = strings.TrimSpace(parts[1])
	}

	fields := strings.Fields(parts[0])
	switch len(fields) {
	case 1:
		target = fields[0]
	case 2:
		source = fields[0]
		target = fields[1]
	case 0:
		err = fmt.Errorf("missing target field")
	default:
		err = fmt.Errorf("too many fields expected at most 2 fields got %d: %s", len(fields), remaining)
	}
	if err == nil && strings.ContainsRune(target, '.') {
		err = fmt.Errorf("the mapping target %q must be a field name but was a path.\nDots \".\" are not allowed.", target)
	}
	return source, target, custom, err
}

func ParseMethodsCfg(ctx *CfgContext, rawConverter *RawConverter, c *Converter) error {
	if c.Type != nil {
		interf := c.Type.Underlying().(*types.Interface)
		for i := 0; i < interf.NumMethods(); i++ {
			fun := interf.Method(i)
			def, err := ParseMethodCfg(ctx, c, fun, rawConverter.Methods[fun.Name()])
			if err != nil {
				return err
			}
			c.Methods = append(c.Methods, def)
		}
		return nil
	}
	for name, lines := range rawConverter.Methods {
		_, fn, err := ctx.Loader.GetOneRaw(c.Package, name)
		if err != nil {
			return err
		}
		def, err := ParseMethodCfg(ctx, c, fn, lines)
		if err != nil {
			return err
		}
		c.Methods = append(c.Methods, def)
	}
	return nil
}

func ParseMethodCfg(ctx *CfgContext, c *Converter, obj types.Object, rawMethod RawLines) (*Method, error) {
	m := &Method{
		Common:      c.Common,
		Fields:      map[string]*FieldMapping{},
		Location:    rawMethod.Location,
		EnumMapping: &EnumMapping{Map: map[string]string{}},
		LocalOpts:   LocalMethodOpts{Context: map[string]bool{}},
	}

	for _, value := range rawMethod.Lines {
		if err := ParseMethodLine(ctx, c, m, value); err != nil {
			return m, FormatLineError(rawMethod, obj.String(), value, err)
		}
	}

	def, err := ParseMethod(obj, &ParseMethodOpts{
		ErrorPrefix:       "error parsing converter method",
		Location:          rawMethod.Location,
		Converter:         nil,
		OutputPackagePath: c.OutputPackagePath,
		Params:            ParamsRequired,
		ContextMatch:      m.ArgContextRegex,
		Generated:         true,
		UpdateParam:       m.UpdateParam,
	}, m.LocalOpts)

	m.MethodDefinition = def

	return m, err
}

func ParseMethodLine(ctx *CfgContext, c *Converter, m *Method, value string) (err error) {
	cmd, rest := ParseCommand(value)
	fieldSetting := false
	switch cmd {
	case ConfigMap:
		fieldSetting = true
		var source, target, custom string
		source, target, custom, err = ParseMethodMap(rest)
		if err != nil {
			return err
		}
		f := m.Field(target)
		f.Source = source

		if custom != "" {
			opts := &ParseMethodOpts{
				ErrorPrefix:       "error parsing type",
				OutputPackagePath: c.OutputPackagePath,
				Converter:         c.TypeForMethod(),
				Params:            ParamsOptional,
				AllowTypeParams:   true,
				ContextMatch:      m.ArgContextRegex,
			}
			f.Function, err = ctx.Loader.GetOne(c.Package, custom, opts)
		}
	case "ignore":
		fieldSetting = true
		fields := strings.Fields(rest)
		for _, f := range fields {
			m.Field(f).Ignore = true
		}
	case "update":
		m.UpdateParam, err = ParseString(rest)
	case "context":
		var key string
		key, err = ParseString(rest)
		m.LocalOpts.Context[key] = true
	case "enum:map":
		fields := strings.Fields(rest)
		if len(fields) != 2 {
			return fmt.Errorf("invalid fields")
		}

		if IsEnumAction(fields[1]) {
			err = ValidateEnumAction(fields[1])
		}

		m.EnumMapping.Map[fields[0]] = fields[1]
	case "enum:transform":
		fields := strings.SplitN(rest, " ", 2)

		config := ""
		if len(fields) == 2 {
			config = fields[1]
		}

		var t ConfiguredTransformer
		t, err = ParseTransformer(ctx, fields[0], config)
		m.EnumMapping.Transformers = append(m.EnumMapping.Transformers, t)
	case "autoMap":
		fieldSetting = true
		var s string
		s, err = ParseString(rest)
		m.AutoMap = append(m.AutoMap, strings.TrimSpace(s))
	case ConfigDefault:
		opts := &ParseMethodOpts{
			ErrorPrefix:       "error parsing type",
			OutputPackagePath: c.OutputPackagePath,
			Converter:         c.TypeForMethod(),
			Params:            ParamsOptional,
			AllowTypeParams:   true,
			ContextMatch:      m.ArgContextRegex,
		}
		m.Constructor, err = ctx.Loader.GetOne(c.Package, rest, opts)
	default:
		fieldSetting, err = ParseCommon(&m.Common, cmd, rest)
	}
	if fieldSetting {
		m.RawFieldSettings = append(m.RawFieldSettings, value)
	}
	return err
}
