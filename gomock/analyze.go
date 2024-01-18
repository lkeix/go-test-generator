package gomock

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GomockComment string

func NewGomockCommnet(comment string) GomockComment {
	return GomockComment(comment)
}

const dstOptionString = "-destination="

func (g GomockComment) DstPath() string {
	splits := strings.Split(string(g), " ")

	for _, s := range splits {
		if strings.HasPrefix(s, dstOptionString) {
			return strings.ReplaceAll(s, dstOptionString, "")
		}
	}

	return ""
}

type Gomock interface {
	ExtractDepsInterface(filePath, funcName string) map[*ast.CallExpr]struct{}
}

type FilePath string

type FuncName string

type gomock struct {
	pkgs      []*packages.Package
	funcDecls map[FilePath]map[FuncName]*ast.FuncDecl
	asts      map[string]*ast.File
}

func (g *gomock) ExtractDepsInterface(filePath, funcName string) map[*ast.CallExpr]struct{} {
	unique := make(map[*ast.CallExpr]struct{})
	for _, pkg := range g.pkgs {
		if slices.Contains(pkg.GoFiles, filePath) {
			callExpr := g.extractInterfaceFunc(pkg, pkg.Syntax, funcName)
			if callExpr != nil {
				unique[callExpr] = struct{}{}
			}
		}
	}
	return unique
}

func (g *gomock) extractInterfaceFunc(pkg *packages.Package, asts []*ast.File, funcName string) *ast.CallExpr {
	for _, a := range asts {
		for _, decl := range a.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				callExpr := g.extractCallExprInInterface(pkg, d.Body)
				if callExpr == nil {
					continue
				}
				return callExpr
			}
		}
	}

	return nil
}

func (g *gomock) extractCallExprInInterface(pkg *packages.Package, bs *ast.BlockStmt) *ast.CallExpr {
	for _, stmt := range bs.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			for _, expr := range s.Rhs {
				if callExpr, ok := expr.(*ast.CallExpr); ok {
					funcObj := extractReference(pkg, callExpr)
					if funcObj == nil {
						continue
					}

					recv := extractReciver(funcObj)
					if recv == nil {
						continue
					}

					if _, ok := recv.Type().Underlying().(*types.Interface); ok {
						return callExpr
					}
				}
			}
		case *ast.ExprStmt:
			callExpr, ok := s.X.(*ast.CallExpr)
			if ok {
				funcObj := extractReference(pkg, callExpr)
				if funcObj == nil {
					continue
				}

				recv := extractReciver(funcObj)
				if recv == nil {
					continue
				}

				if _, ok := recv.Type().Underlying().(*types.Interface); ok {
					return callExpr
				}
			}
		}
	}
	return nil
}

func extractReference(pkg *packages.Package, callExpr *ast.CallExpr) types.Object {
	switch f := callExpr.Fun.(type) {
	case *ast.Ident:
		return pkg.TypesInfo.ObjectOf(f)
	case *ast.SelectorExpr:
		return pkg.TypesInfo.ObjectOf(f.Sel)
	}
	return nil
}

func extractReciver(funcObj types.Object) *types.Var {
	if sig, ok := funcObj.Type().(*types.Signature); ok {
		if recv := sig.Recv(); recv != nil {
				return recv
		}
	}
	return nil
}

func NewGomock(rootDir string, asts map[string]*ast.File) (Gomock, error) {
	funcDecls := make(map[FilePath]map[FuncName]*ast.FuncDecl)
	for path, ast := range asts {
		fp := FilePath(path)
		f := extractFuncs(ast.Decls)
		funcDecls[fp] = f
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir: rootDir,
	}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("%s/...", rootDir))
	if err != nil {
		return nil, err
	}

	return &gomock{
		pkgs:      pkgs,
		asts:      asts,
		funcDecls: funcDecls,
	}, nil
}

func extractFuncs(decls []ast.Decl) map[FuncName]*ast.FuncDecl {
	ret := make(map[FuncName]*ast.FuncDecl)
	for _, decl := range decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			ret[FuncName(d.Name.Name)] = d
		}
	}

	return ret
}
