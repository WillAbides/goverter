package goverter

import (
	"fmt"
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
