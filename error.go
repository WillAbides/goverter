package goverter

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/dave/jennifer/jen"
)

// errorMessagePath defines the path inside an error message.
type errorMessagePath struct {
	Prefix     string
	SourceID   string
	TargetID   string
	SourceType string
	TargetType string
}

// buildError defines a conversion error.
type buildError struct {
	Path  []*errorMessagePath
	Cause string
}

// bewBuildError creates an error.
func bewBuildError(cause string) *buildError {
	return &buildError{Cause: cause, Path: []*errorMessagePath{}}
}

// Lift appends the path to the error.
func (e *buildError) Lift(paths ...*errorMessagePath) *buildError {
	e.Path = append(paths, e.Path...)
	return e
}

// buildErrorToString converts the error into a string.
func buildErrorToString(err *buildError) string {
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

type errorPath []errorElement

func (e errorPath) WrapErrors(errStmt *jen.Statement) *jen.Statement {
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

func (e errorPath) WrapErrorsUsing(pkg string, errStmt *jen.Statement) *jen.Statement {
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

func (e errorPath) Index(code *jen.Statement) errorPath { return append(e, errElmIndex{code}) }
func (e errorPath) Key(code *jen.Statement) errorPath   { return append(e, errElmKey{code}) }
func (e errorPath) Field(name string) errorPath         { return append(e, errElmField(name)) }

type errorElement interface{ _elm() }

type (
	errElmIndex struct{ stmt *jen.Statement }
	errElmKey   struct{ stmt *jen.Statement }
	errElmField string
)

func (errElmKey) _elm()   {}
func (errElmIndex) _elm() {}
func (errElmField) _elm() {}
