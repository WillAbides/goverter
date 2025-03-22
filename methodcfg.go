package goverter

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
