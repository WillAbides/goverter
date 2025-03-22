package goverter

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
