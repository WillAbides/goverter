package goverter

import (
	"regexp"
)

type Common struct {
	FieldSettings                      []string
	WrapErrors                         bool
	WrapErrorsUsing                    string
	IgnoreUnexported                   bool
	IgnoreBasicZeroValueField          bool
	IgnoreStructZeroValueField         bool
	IgnoreNillableZeroValueField       bool
	MatchIgnoreCase                    bool
	IgnoreMissing                      bool
	SkipCopySameType                   bool
	UseZeroValueOnPointerInconsistency bool
	UseUnderlyingTypeMethods           bool
	DefaultUpdate                      bool
	ArgContextRegex                    *regexp.Regexp
	Enum                               EnumConfig
}
