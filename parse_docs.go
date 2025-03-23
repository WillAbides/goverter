package goverter

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

const (
	converterMarker = directivePrefix + "converter"
)

// parseDocsConfig provides input to the parseDocs method below.
type parseDocsConfig struct {
	// PackagePatterns are golang package patterns to scan, required.
	PackagePattern []string
	// WorkingDir is a directory to invoke the tool on. If omitted, current directory is used.
	WorkingDir string
	BuildTags  string
}

// parseDocs parses the docs for the given pattern.
func parseDocs(c parseDocsConfig) ([]rawConverter, error) {
	loadCfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  c.WorkingDir,
	}
	if c.BuildTags != "" {
		loadCfg.BuildFlags = append(loadCfg.BuildFlags, "-tags", c.BuildTags)
	}
	pkgs, err := packages.Load(loadCfg, c.PackagePattern...)
	if err != nil {
		return nil, err
	}
	rawConverters := []rawConverter{}
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf(`could not load package %s

%s

Goverter cannot generate converters when there are compile errors because it
requires the type information from the compiled sources.`, pkg.PkgPath, pkg.Errors[0])
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					converters, err := parseGenDecl(pkg, genDecl)
					if err != nil {
						location := pkg.Fset.Position(genDecl.Pos()).String()
						return rawConverters, fmt.Errorf("%s: %s", location, err)
					}
					rawConverters = append(rawConverters, converters...)
				}
			}
		}
	}
	return rawConverters, nil
}

func parseFunctions(pkg *packages.Package, decl *ast.GenDecl, lines rawLines) ([]rawConverter, error) {
	if decl.Tok != token.VAR {
		return nil, fmt.Errorf("%s must be defined on %q-block but was %q", converterMarker, token.VAR, decl.Tok.String())
	}

	location := pkg.Fset.Position(decl.Pos())

	result := map[string]rawLines{}
	for _, spec := range decl.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			return nil, fmt.Errorf("expected value spec but got %#v", spec)
		}
		if len(value.Names) != 1 {
			return nil, fmt.Errorf("must have one name")
		}
		name := value.Names[0].Name
		result[name] = rawLinesForNode(pkg, value)
	}

	converter := rawConverter{
		FileName:    location.Filename,
		Converter:   lines,
		Methods:     result,
		PackageName: pkg.Name,
		PackagePath: pkg.PkgPath,
	}
	return []rawConverter{converter}, nil
}

func parseGenDecl(pkg *packages.Package, decl *ast.GenDecl) ([]rawConverter, error) {
	lines := rawLinesForNode(pkg, decl)

	if lines.hasSetting("variables") {
		return parseFunctions(pkg, decl, lines)
	}

	if lines.hasSetting("converter") {
		if decl.Tok != token.TYPE {
			return nil, fmt.Errorf("%s must be defined on %q-block but was %q", converterMarker, token.TYPE, decl.Tok.String())
		}

		if len(decl.Specs) != 1 {
			return nil, fmt.Errorf("found %s on type but it has multiple interfaces inside", converterMarker)
		}
		typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
		if !ok {
			return nil, fmt.Errorf("%s may only be applied to type declarations ", converterMarker)
		}
		c, err := parseInterface(pkg, typeSpec, rawLines{
			Lines:    lines.Lines,
			Location: nodeLocation(pkg.Fset, typeSpec),
		})
		if err != nil {
			return nil, err
		}
		return []rawConverter{c}, nil
	}

	var converters []rawConverter

	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		tsLines := rawLinesForNode(pkg, typeSpec)
		if tsLines.hasSetting("converter") {
			c, err := parseInterface(pkg, typeSpec, tsLines)
			if err != nil {
				return nil, err
			}
			converters = append(converters, c)
		}
	}

	return converters, nil
}

func parseInterface(pkg *packages.Package, typeSpec *ast.TypeSpec, lines rawLines) (rawConverter, error) {
	astInterface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		return rawConverter{}, fmt.Errorf("%s may only be applied to type interface declarations ", converterMarker)
	}
	typeName := typeSpec.Name.String()

	location := pkg.Fset.Position(typeSpec.Pos())
	methods, err := parseInterfaceMethods(pkg, astInterface)
	if err != nil {
		return rawConverter{}, fmt.Errorf("type %s: %s", typeName, err)
	}
	converter := rawConverter{
		InterfaceName: typeName,
		FileName:      location.Filename,
		Converter:     lines,
		Methods:       methods,
		PackageName:   pkg.Name,
		PackagePath:   pkg.PkgPath,
	}
	return converter, nil
}

func parseInterfaceMethods(pkg *packages.Package, inter *ast.InterfaceType) (map[string]rawLines, error) {
	result := map[string]rawLines{}
	for _, method := range inter.Methods.List {
		if len(method.Names) != 1 {
			return result, fmt.Errorf("method must have one name")
		}
		name := method.Names[0].String()
		result[name] = rawLinesForNode(pkg, method)
	}
	return result, nil
}

func rawLinesForNode(pkg *packages.Package, node ast.Node) rawLines {
	var comments []*ast.CommentGroup
	for _, file := range pkg.Syntax {
		if pkg.Fset.File(file.Pos()) == pkg.Fset.File(node.Pos()) {
			comments = ast.NewCommentMap(pkg.Fset, file, file.Comments)[node]
			break
		}
	}
	return rawLines{
		Location: nodeLocation(pkg.Fset, node),
		Lines:    commentGroupSettingLines(comments),
	}
}

func nodeLocation(fset *token.FileSet, node ast.Node) string {
	p := fset.Position(node.Pos())
	return fmt.Sprintf("%s:%d", p.Filename, p.Line)
}
