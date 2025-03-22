package goverter

import (
	"go/types"
)

// Accessible checks if obj is accessible within outputPackagePath.
func Accessible(obj types.Object, outputPackagePath string) bool {
	if obj.Exported() {
		return true
	}

	pkg := obj.Pkg()
	return pkg == nil || pkg.Path() == outputPackagePath
}

// Signature represents a signature for conversion.
type Signature struct {
	Source string
	Target string
}
