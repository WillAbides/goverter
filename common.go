package goverter

import (
	"fmt"
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

func ParseCommon(c *Common, cmd, rest string) (fieldSetting bool, err error) {
	switch cmd {
	case "wrapErrors":
		if c.WrapErrorsUsing != "" {
			return false, fmt.Errorf("cannot be used in combination with wrapErrorsUsing")
		}
		c.WrapErrors, err = ParseBool(rest)
	case "wrapErrorsUsing":
		if c.WrapErrors {
			return false, fmt.Errorf("cannot be used in combination with wrapErrors")
		}
		c.WrapErrorsUsing, err = ParseString(rest)
	case "ignoreUnexported":
		fieldSetting = true
		c.IgnoreUnexported, err = ParseBool(rest)
	case "update:ignoreZeroValueField":
		fieldSetting = true
		c.IgnoreBasicZeroValueField, err = ParseBool(rest)
		c.IgnoreStructZeroValueField = c.IgnoreBasicZeroValueField
		c.IgnoreNillableZeroValueField = c.IgnoreBasicZeroValueField
	case "update:ignoreZeroValueField:basic":
		c.IgnoreBasicZeroValueField, err = ParseBool(rest)
	case "update:ignoreZeroValueField:struct":
		c.IgnoreStructZeroValueField, err = ParseBool(rest)
	case "update:ignoreZeroValueField:nillable":
		c.IgnoreNillableZeroValueField, err = ParseBool(rest)
	case "default:update":
		c.DefaultUpdate, err = ParseBool(rest)
	case "matchIgnoreCase":
		fieldSetting = true
		c.MatchIgnoreCase, err = ParseBool(rest)
	case "ignoreMissing":
		fieldSetting = true
		c.IgnoreMissing, err = ParseBool(rest)
	case "skipCopySameType":
		c.SkipCopySameType, err = ParseBool(rest)
	case "useZeroValueOnPointerInconsistency":
		c.UseZeroValueOnPointerInconsistency, err = ParseBool(rest)
	case "useUnderlyingTypeMethods":
		c.UseUnderlyingTypeMethods, err = ParseBool(rest)
	case "enum":
		c.Enum.Enabled, err = ParseBool(rest)
	case "arg:context:regex":
		c.ArgContextRegex, err = ParseRegex(rest)
	case "enum:unknown":
		c.Enum.Unknown, err = ParseString(rest)
		if err == nil && IsEnumAction(c.Enum.Unknown) {
			err = ValidateEnumAction(c.Enum.Unknown)
		}
	case "":
		err = fmt.Errorf("missing setting key")
	default:
		err = fmt.Errorf("unknown setting: %s", cmd)
	}

	return fieldSetting, err
}
