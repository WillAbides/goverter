package config

import (
	"fmt"

	"github.com/jmattheis/goverter"
)

func parseCommon(c *goverter.Common, cmd, rest string) (fieldSetting bool, err error) {
	switch cmd {
	case "wrapErrors":
		if c.WrapErrorsUsing != "" {
			return false, fmt.Errorf("cannot be used in combination with wrapErrorsUsing")
		}
		c.WrapErrors, err = goverter.ParseBool(rest)
	case "wrapErrorsUsing":
		if c.WrapErrors {
			return false, fmt.Errorf("cannot be used in combination with wrapErrors")
		}
		c.WrapErrorsUsing, err = goverter.ParseString(rest)
	case "ignoreUnexported":
		fieldSetting = true
		c.IgnoreUnexported, err = goverter.ParseBool(rest)
	case "update:ignoreZeroValueField":
		fieldSetting = true
		c.IgnoreBasicZeroValueField, err = goverter.ParseBool(rest)
		c.IgnoreStructZeroValueField = c.IgnoreBasicZeroValueField
		c.IgnoreNillableZeroValueField = c.IgnoreBasicZeroValueField
	case "update:ignoreZeroValueField:basic":
		c.IgnoreBasicZeroValueField, err = goverter.ParseBool(rest)
	case "update:ignoreZeroValueField:struct":
		c.IgnoreStructZeroValueField, err = goverter.ParseBool(rest)
	case "update:ignoreZeroValueField:nillable":
		c.IgnoreNillableZeroValueField, err = goverter.ParseBool(rest)
	case "default:update":
		c.DefaultUpdate, err = goverter.ParseBool(rest)
	case "matchIgnoreCase":
		fieldSetting = true
		c.MatchIgnoreCase, err = goverter.ParseBool(rest)
	case "ignoreMissing":
		fieldSetting = true
		c.IgnoreMissing, err = goverter.ParseBool(rest)
	case "skipCopySameType":
		c.SkipCopySameType, err = goverter.ParseBool(rest)
	case "useZeroValueOnPointerInconsistency":
		c.UseZeroValueOnPointerInconsistency, err = goverter.ParseBool(rest)
	case "useUnderlyingTypeMethods":
		c.UseUnderlyingTypeMethods, err = goverter.ParseBool(rest)
	case "enum":
		c.Enum.Enabled, err = goverter.ParseBool(rest)
	case "arg:context:regex":
		c.ArgContextRegex, err = goverter.ParseRegex(rest)
	case "enum:unknown":
		c.Enum.Unknown, err = goverter.ParseString(rest)
		if err == nil && IsEnumAction(c.Enum.Unknown) {
			err = validateEnumAction(c.Enum.Unknown)
		}
	case "":
		err = fmt.Errorf("missing setting key")
	default:
		err = fmt.Errorf("unknown setting: %s", cmd)
	}

	return fieldSetting, err
}
