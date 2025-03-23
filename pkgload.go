package goverter

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

func newPackageLoader(workDir, buildTags string, paths []string) (*packageLoader, error) {
	loader := &packageLoader{
		lookup: map[string]*packages.Package{},
		locals: map[string]map[string]localMethodOpts{},
	}
	err := loader.Load(workDir, buildTags, paths)
	return loader, err
}

type packageLoader struct {
	lookup map[string]*packages.Package
	locals map[string]map[string]localMethodOpts
}

func (g *packageLoader) GetMatching(cwd, fullMethod string, opts *parseMethodOpts) ([]*methodDefinition, error) {
	pkgName, name, err := parseMethodString(cwd, fullMethod)
	if err != nil {
		return nil, err
	}

	pattern, err := regexp.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("could not parse name as regexp %q: %s", name, err)
	}

	if _, complete := pattern.LiteralPrefix(); complete {
		// not a regex
		m, err := g.GetOneParsed(pkgName, name, opts)
		if err != nil {
			return nil, err
		}
		return []*methodDefinition{m}, nil
	}

	// this is regexp, scan thru the package methods to find funcs that match the pattern
	var matches []*methodDefinition

	pkg, err := g.GetPkg(pkgName)
	if err != nil {
		return nil, err
	}

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		loc := pattern.FindStringIndex(name)
		if len(loc) != 2 {
			continue
		}
		if loc[0] != 0 || loc[1] != len(name) {
			// we want full match only: e.g. CopyAbc.* won't match OtherCopyAbc
			continue
		}

		obj := scope.Lookup(name)
		m, err := parseMethod(obj, opts, g.LocalConfig(pkg, name))
		if err == nil {
			matches = append(matches, m)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf(`package %s does not have methods with names that match
the golang regexp pattern %q and a convert signature`, pkgName, name)
	}

	return matches, nil
}

func (g *packageLoader) GetUncheckedPkg(pkgName string) *packages.Package {
	return g.lookup[pkgName]
}

func (g *packageLoader) GetPkg(pkgName string) (*packages.Package, error) {
	pkg := g.lookup[pkgName]
	if pkg == nil {
		return nil, fmt.Errorf("failed to load package %q:\nmake sure it's a valid golang package", pkgName)
	}

	if len(pkg.Errors) > 0 {
		var lines []string
		for _, err := range pkg.Errors {
			lines = append(lines, err.Error())
		}

		return nil, fmt.Errorf("failed to load package %q:\n%s", pkgName, strings.Join(lines, "\n"))
	}
	return pkg, nil
}

func (g *packageLoader) LocalConfig(pkg *packages.Package, name string) localMethodOpts {
	fns, ok := g.locals[pkg.PkgPath]
	if !ok {
		fns = map[string]localMethodOpts{}
		for _, file := range pkg.Syntax {
			commentMap := ast.NewCommentMap(pkg.Fset, file, file.Comments)
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					lines := commentGroupSettingLines(commentMap[fn])
					if len(lines) == 0 {
						continue
					}

					contexts := map[string]bool{}
					for _, line := range lines {
						if cmd, rest := parseCommand(line); cmd == "context" {
							if ctx, err := parseString(rest); err == nil {
								contexts[ctx] = true
							}
						}
					}
					fns[fn.Name.Name] = localMethodOpts{Context: contexts}
				}
			}
		}
		g.locals[pkg.PkgPath] = fns
	}
	fn, ok := fns[name]
	if !ok {
		return emptyLocalMethodOpts
	}
	return fn
}

func (g *packageLoader) GetOneRaw(pkgName, name string) (*packages.Package, types.Object, error) {
	pkg, err := g.GetPkg(pkgName)
	if err != nil {
		return pkg, nil, err
	}

	obj := pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return pkg, nil, fmt.Errorf("%q does not exist in package %q", name, pkgName)
	}
	return pkg, obj, nil
}

func (g *packageLoader) GetOne(sourcePackage, fullMethod string, opts *parseMethodOpts) (*methodDefinition, error) {
	pkgName, name, err := parseMethodString(sourcePackage, fullMethod)
	if err != nil {
		return nil, err
	}
	return g.GetOneParsed(pkgName, name, opts)
}

func (g *packageLoader) GetOneParsed(pkgName, name string, opts *parseMethodOpts) (*methodDefinition, error) {
	pkg, obj, err := g.GetOneRaw(pkgName, name)
	if err != nil {
		return nil, err
	}

	def, err := parseMethod(obj, opts, g.LocalConfig(pkg, name))
	if err != nil {
		return nil, err
	}
	return def, nil
}

// Load is used to load extend packages, with caching support.
func (g *packageLoader) Load(workDir, buildTags string, paths []string) error {
	packagesCfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  workDir,
	}
	if buildTags != "" {
		packagesCfg.BuildFlags = append(packagesCfg.BuildFlags, "-tags", buildTags)
	}
	pkgs, err := packages.Load(packagesCfg, paths...)
	if err != nil {
		// This happens rare, and only if somebody uses advanced package pattern query in a wrong way.
		// The cause (err) usually has enough details to troubleshoot this issue.
		return fmt.Errorf("failed to load packages %s:\n%s", paths, err)
	}
	for _, pkg := range pkgs {
		g.lookup[pkg.PkgPath] = pkg
	}

	return nil
}
