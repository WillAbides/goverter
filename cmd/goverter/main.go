package main

import (
	"os"

	"github.com/jmattheis/goverter"
)

func main() {
	goverter.Run(os.Args, goverter.RunOpts{})
}
