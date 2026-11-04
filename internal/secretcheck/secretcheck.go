// Package secretcheck reports exported struct fields, and parameters and
// results of exported functions and interface methods, that are named like a
// secret but hold a plain string or []byte instead of a secret.String.
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
				checkTypes(pass, decl)
			}
		}
	}

	return nil, nil //nolint:nilnil // An analyzer without facts returns neither.
}

func checkFunc(pass *analysis.Pass, decl *ast.FuncDecl) {
	if !decl.Name.IsExported() {
		return
	}

	checkSignature(pass, decl.Type, decl.Name.Name)
}

func checkSignature(pass *analysis.Pass, funcType *ast.FuncType, funcName string) {
	for _, list := range []*ast.FieldList{funcType.Params, funcType.Results} {
		for _, field := range fields(list) {
			for _, name := range field.Names {
				report(pass, field.Type, name, funcName+"'s "+name.Name)
			}
		}
	}
}

func checkTypes(pass *analysis.Pass, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !typeSpec.Name.IsExported() {
			continue
		}

		switch typ := typeSpec.Type.(type) {
		case *ast.StructType:
			checkStruct(pass, typeSpec.Name.Name, typ)
		case *ast.InterfaceType:
			checkInterface(pass, typeSpec.Name.Name, typ)
		}
	}
}

func checkStruct(pass *analysis.Pass, typeName string, typ *ast.StructType) {
	for _, field := range fields(typ.Fields) {
		for _, name := range field.Names {
			if name.IsExported() {
				report(pass, field.Type, name, typeName+"."+name.Name)
			}
		}
	}
}

func checkInterface(pass *analysis.Pass, typeName string, typ *ast.InterfaceType) {
	for _, method := range fields(typ.Methods) {
		// An embedded interface has no name.
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		name := method.Names[0]
		if name.IsExported() {
			checkSignature(pass, funcType, typeName+"."+name.Name)
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
