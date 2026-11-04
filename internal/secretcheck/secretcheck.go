// Package secretcheck reports exported struct fields, parameters and results
// that are named like a secret but hold a plain string or []byte instead of
// a secret.String.
package secretcheck

import (
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports secrets in exported APIs that aren't a secret.String.
var Analyzer = &analysis.Analyzer{
	Name: "secretcheck",
	Doc:  "report exported fields, parameters and results named like a secret that aren't a secret.String",
	Run:  run,
}

var secretName = regexp.MustCompile(`(?i)password|passwd|plaintext|secret|token|apikey|privatekey|credential`)

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		name := pass.Fset.File(file.Pos()).Name()
		if strings.HasSuffix(name, "_test.go") || ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				checkFunc(pass, decl)
			case *ast.GenDecl:
				checkStructs(pass, decl)
			}
		}
	}

	return nil, nil //nolint:nilnil // An analyzer without facts returns neither.
}

func checkFunc(pass *analysis.Pass, decl *ast.FuncDecl) {
	if !decl.Name.IsExported() {
		return
	}

	for _, list := range []*ast.FieldList{decl.Type.Params, decl.Type.Results} {
		for _, field := range fields(list) {
			for _, name := range field.Names {
				report(pass, field.Type, name, decl.Name.Name+"'s "+name.Name)
			}
		}
	}
}

func checkStructs(pass *analysis.Pass, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !typeSpec.Name.IsExported() {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range fields(structType.Fields) {
			for _, name := range field.Names {
				if name.IsExported() {
					report(pass, field.Type, name, typeSpec.Name.Name+"."+name.Name)
				}
			}
		}
	}
}

func fields(list *ast.FieldList) []*ast.Field {
	if list == nil {
		return nil
	}

	return list.List
}

func report(pass *analysis.Pass, typeExpr ast.Expr, name *ast.Ident, what string) {
	if !secretName.MatchString(name.Name) {
		return
	}

	typ := pass.TypesInfo.TypeOf(typeExpr)
	if typ == nil || !plain(typ) {
		return
	}

	pass.Reportf(name.Pos(), "%s holds a secret as %s, use secret.String",
		what, types.TypeString(typ, types.RelativeTo(pass.Pkg)))
}

// plain reports whether typ is a string or a []byte, also behind a named
// type.
func plain(typ types.Type) bool {
	switch underlying := typ.Underlying().(type) {
	case *types.Basic:
		return underlying.Kind() == types.String
	case *types.Slice:
		elem, ok := underlying.Elem().Underlying().(*types.Basic)

		return ok && elem.Kind() == types.Byte
	default:
		return false
	}
}
