package goverter

type ArgUse string

const (
	ArgUseSource      ArgUse = "source"
	ArgUseMultiSource ArgUse = "additional-source"
	ArgUseInterface   ArgUse = "interface"
	ArgUseContext     ArgUse = "context"
	ArgUseTarget      ArgUse = "target"
)

type Arg struct {
	Name string
	Use  ArgUse
	Type *Type
}

type Parameters struct {
	TypeParams bool

	Source       *Type
	MultiSources []*Type
	Target       *Type
	Context      map[string]*Type

	Signature Signature

	RawArgs []Arg

	ReturnError  bool
	UpdateTarget bool
}
