package goverter

import (
	"fmt"
	"go/types"
	"strings"
)

const (
	configMap     = "map"
	configDefault = "default"
)

type fieldMapping struct {
	Source   string
	Function *methodDefinition
	Ignore   bool
}

type method struct {
	*methodDefinition
	commonCfg

	Constructor *methodDefinition
	AutoMap     []string
	Fields      map[string]*fieldMapping
	EnumMapping *EnumMapping

	RawFieldSettings []string

	Location    string
	UpdateParam string
	LocalOpts   localMethodOpts
}

func (m *method) Field(targetName string) *fieldMapping {
	target, ok := m.Fields[targetName]
	if !ok {
		target = &fieldMapping{}
		m.Fields[targetName] = target
	}
	return target
}

func parseMethodMap(remaining string) (source, target, custom string, err error) {
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

func parseMethodsCfg(ctx *CfgContext, rawConverter *RawConverter, c *Converter) error {
	if c.Type != nil {
		interf := c.Type.Underlying().(*types.Interface)
		for i := 0; i < interf.NumMethods(); i++ {
			fun := interf.Method(i)
			def, err := parseMethodCfg(ctx, c, fun, rawConverter.Methods[fun.Name()])
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
		def, err := parseMethodCfg(ctx, c, fn, lines)
		if err != nil {
			return err
		}
		c.Methods = append(c.Methods, def)
	}
	return nil
}

func parseMethodCfg(ctx *CfgContext, c *Converter, obj types.Object, rawMethod RawLines) (*method, error) {
	m := &method{
		commonCfg:   c.commonCfg,
		Fields:      map[string]*fieldMapping{},
		Location:    rawMethod.Location,
		EnumMapping: &EnumMapping{Map: map[string]string{}},
		LocalOpts:   localMethodOpts{Context: map[string]bool{}},
	}

	for _, value := range rawMethod.Lines {
		if err := parseMethodLine(ctx, c, m, value); err != nil {
			return m, FormatLineError(rawMethod, obj.String(), value, err)
		}
	}

	def, err := parseMethod(obj, &parseMethodOpts{
		ErrorPrefix:       "error parsing converter method",
		Location:          rawMethod.Location,
		Converter:         nil,
		OutputPackagePath: c.OutputPackagePath,
		Params:            paramsRequired,
		ContextMatch:      m.ArgContextRegex,
		Generated:         true,
		UpdateParam:       m.UpdateParam,
	}, m.LocalOpts)

	m.methodDefinition = def

	return m, err
}

func parseMethodLine(ctx *CfgContext, c *Converter, m *method, value string) (err error) {
	cmd, rest := parseCommand(value)
	fieldSetting := false
	switch cmd {
	case configMap:
		fieldSetting = true
		var source, target, custom string
		source, target, custom, err = parseMethodMap(rest)
		if err != nil {
			return err
		}
		f := m.Field(target)
		f.Source = source

		if custom != "" {
			opts := &parseMethodOpts{
				ErrorPrefix:       "error parsing type",
				OutputPackagePath: c.OutputPackagePath,
				Converter:         c.typeForMethod(),
				Params:            paramsOptional,
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
		m.UpdateParam, err = parseString(rest)
	case "context":
		var key string
		key, err = parseString(rest)
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
		s, err = parseString(rest)
		m.AutoMap = append(m.AutoMap, strings.TrimSpace(s))
	case configDefault:
		opts := &parseMethodOpts{
			ErrorPrefix:       "error parsing type",
			OutputPackagePath: c.OutputPackagePath,
			Converter:         c.typeForMethod(),
			Params:            paramsOptional,
			AllowTypeParams:   true,
			ContextMatch:      m.ArgContextRegex,
		}
		m.Constructor, err = ctx.Loader.GetOne(c.Package, rest, opts)
	default:
		fieldSetting, err = parseCommon(&m.commonCfg, cmd, rest)
	}
	if fieldSetting {
		m.RawFieldSettings = append(m.RawFieldSettings, value)
	}
	return err
}
