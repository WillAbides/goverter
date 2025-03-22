package cli

type Command interface {
	_c()
}

type Generate struct {
	Config *GenerateConfig
}

type Help struct {
	Usage string
}

type Version struct{}

func (*Help) _c()     {}
func (*Generate) _c() {}
func (*Version) _c()  {}
