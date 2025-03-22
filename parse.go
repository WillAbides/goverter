package goverter

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func parseMethodString(sourcePackage, fullMethod string) (pkg, name string, err error) {
	parts := strings.SplitN(fullMethod, ":", 2)
	switch len(parts) {
	case 0:
		return pkg, name, fmt.Errorf("invalid custom method: %s", fullMethod)
	case 1:
		name = parts[0]
		pkg = sourcePackage
	case 2:
		pkg = parts[0]
		name = parts[1]
		if strings.HasPrefix(pkg, "../") || strings.HasPrefix(pkg, "./") || pkg == "." {
			pkg = path.Join(sourcePackage, pkg)
		}

		if pkg == "" {
			// example: goverter:extend :MyLocalConvert
			// the purpose of the ':' in this case is confusing, do not allow such case
			return pkg, name, fmt.Errorf(`package path must not be empty in the custom method %q.
See https://goverter.jmattheis.de/reference/extend`, fullMethod)
		}
	}

	if name == "" {
		return pkg, name, fmt.Errorf(`method name pattern is required in the custom method %q.
See https://goverter.jmattheis.de/reference/extend`, fullMethod)
	}
	if strings.Contains(pkg, "...") {
		return pkg, name, fmt.Errorf(`package wildcard pattern ... is not supported: %q`, fullMethod)
	}
	return pkg, name, nil
}

func parseCommand(value string) (string, string) {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func parseEnum[T ~string](empty bool, remaining string, values ...T) (T, error) {
	fields := strings.Fields(remaining)

	switch {
	case len(fields) == 0 && empty:
		return "", nil
	case len(fields) == 1:
		for _, value := range values {
			if fields[0] == string(value) {
				return value, nil
			}
		}

		return "", fmt.Errorf("invalid value: '%s' must be one of: %s", fields[0], formatValues(values))
	default:
		return "", fmt.Errorf("invalid value: expected one value but got %d: %s", len(fields), fields)
	}
}

func formatValues[T ~string](values []T) string {
	strs := make([]string, len(values))
	for i, id := range values {
		strs[i] = string(id)
	}
	return strings.Join(strs, ", ")
}

func parseBool(remaining string) (bool, error) {
	val, err := parseEnum(true, remaining, "yes", "no")
	return val == "" || val == "yes", err
}

func parseString(remaining string) (string, error) {
	fields := strings.Fields(remaining)
	if len(fields) != 1 {
		return "", fmt.Errorf("must have one value but got %d: %#v", len(fields), remaining)
	}
	return fields[0], nil
}

func parseRegex(remaining string) (*regexp.Regexp, error) {
	value, err := parseString(remaining)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(value)
}

func parseFile(cwd, rest string) (string, error) {
	field, err := parseString(rest)
	if err != nil {
		return field, err
	}

	if strings.HasPrefix(field, "@cwd/") {
		return filepath.Abs(filepath.Join(cwd, strings.TrimPrefix(field, "@cwd/")))
	}
	return field, nil
}
