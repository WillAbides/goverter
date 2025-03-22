package goverter

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/dave/jennifer/jen"
)

// ErrorMessagePath defines the path inside an error message.
type ErrorMessagePath struct {
	Prefix     string
	SourceID   string
	TargetID   string
	SourceType string
	TargetType string
}

// BuildError defines a conversion error.
type BuildError struct {
	Path  []*ErrorMessagePath
	Cause string
}

// NewBuildError creates an error.
func NewBuildError(cause string) *BuildError {
	return &BuildError{Cause: cause, Path: []*ErrorMessagePath{}}
}

// Lift appends the path to the error.
func (e *BuildError) Lift(paths ...*ErrorMessagePath) *BuildError {
	e.Path = append(paths, e.Path...)
	return e
}

// BuildErrorToString converts the error into a string.
func BuildErrorToString(err *BuildError) string {
	if len(err.Path) == 0 {
		panic("oops that shouldn't happen")
	}

	sourcePaths := 0
	targetPaths := 0
	for _, path := range err.Path {
		if path.SourceType != "" {
			sourcePaths++
		}
		if path.TargetType != "" {
			targetPaths++
		}
	}

	end := 2 + (sourcePaths+targetPaths)*2 - 1
	sourceLine := (sourcePaths * 2)
	targetLine := sourceLine + 1

	lines := make([]string, end+1)

	sourceTypeLine := 0
	targetTypeLine := end
	for i := 0; i < len(err.Path); i++ {
		path := err.Path[i]
		padding := int(math.Max(float64(len(path.SourceID)), float64(len(path.TargetID))))

		if path.SourceType != "" {
			lines[sourceTypeLine] += strings.Repeat(" ", len(path.Prefix)) + "| " + path.SourceType

			for j := sourceTypeLine + 1; j < sourceLine; j++ {
				lines[j] += strings.Repeat(" ", len(path.Prefix)) + "|" + strings.Repeat(" ", padding-1)
			}
			sourceTypeLine += 2
		} else {
			for j := sourceTypeLine; j < sourceLine; j++ {
				lines[j] += strings.Repeat(" ", len(path.Prefix)+padding)
			}
		}

		lines[sourceLine] += path.Prefix + path.SourceID + strings.Repeat(" ", padding-len(path.SourceID))

		if path.TargetType != "" {
			lines[targetLine] += path.Prefix + path.TargetID + strings.Repeat(" ", padding-len(path.TargetID))

			for j := targetTypeLine - 1; j > targetLine; j-- {
				lines[j] += strings.Repeat(" ", len(path.Prefix)) + "|" + strings.Repeat(" ", padding-1)
			}
			lines[targetTypeLine] += strings.Repeat(" ", len(path.Prefix)) + "| " + path.TargetType
			targetTypeLine -= 2
		} else {
			for j := targetTypeLine; j >= targetLine; j-- {
				lines[j] += strings.Repeat(" ", len(path.Prefix)+padding)
			}
		}
	}

	buf := bytes.Buffer{}
	for _, line := range lines {
		_, _ = fmt.Fprintln(&buf, strings.TrimSpace(line))
	}
	fmt.Fprintln(&buf)
	fmt.Fprint(&buf, err.Cause)
	return buf.String()
}

type ErrorPath []ErrorElement

func (e ErrorPath) WrapErrors(errStmt *jen.Statement) *jen.Statement {
	if len(e) != 0 {
		switch elm := e[len(e)-1].(type) {
		case errElmField:
			return jen.Qual("fmt", "Errorf").Call(jen.Lit("error setting field "+string(elm)+": %w"), errStmt)
		case errElmIndex:
			return jen.Qual("fmt", "Errorf").Call(jen.Lit("error setting index %d: %w"), elm.stmt.Clone(), errStmt)
		}
	}
	return errStmt
}

func (e ErrorPath) WrapErrorsUsing(pkg string, errStmt *jen.Statement) *jen.Statement {
	var args []jen.Code
	for _, elm := range e {
		switch elm := elm.(type) {
		case errElmField:
			args = append(args, jen.Qual(pkg, "Field").Call(jen.Lit(string(elm))))
		case errElmIndex:
			args = append(args, jen.Qual(pkg, "Index").Call(elm.stmt.Clone()))
		case errElmKey:
			args = append(args, jen.Qual(pkg, "Key").Call(elm.stmt.Clone()))
		}
	}
	args = append([]jen.Code{errStmt}, args...)

	return jen.Qual(pkg, "Wrap").Call(args...)
}

func (e ErrorPath) Index(code *jen.Statement) ErrorPath { return append(e, errElmIndex{code}) }
func (e ErrorPath) Key(code *jen.Statement) ErrorPath   { return append(e, errElmKey{code}) }
func (e ErrorPath) Field(name string) ErrorPath         { return append(e, errElmField(name)) }

type ErrorElement interface{ _elm() }

type (
	errElmIndex struct{ stmt *jen.Statement }
	errElmKey   struct{ stmt *jen.Statement }
	errElmField string
)

func (errElmKey) _elm()   {}
func (errElmIndex) _elm() {}
func (errElmField) _elm() {}
