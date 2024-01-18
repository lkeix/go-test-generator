package gomock

import (
	"fmt"
	"go/ast"
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
	ExtractDepsInterface(filePath, funcName string)
}

type FilePath string

type FuncName string

type gomock struct {
	pkgs      []*packages.Package
	funcDecls map[FilePath]map[FuncName]*ast.FuncDecl
	asts      map[string]*ast.File
}

func (g *gomock) ExtractDepsInterface(filePath, funcName string) {
	for _, pkg := range g.pkgs {
		if slices.Contains(pkg.GoFiles, filePath) {
			g.extractInterfaceFunc(pkg, pkg.Syntax, funcName)
		}
	}

	/*
		for _, block := range g.Body.List {
			switch d := block.(type) {
			case *ast.ExprStmt:
				f, ok := d.X.(*ast.CallExpr)
				if ok {
					g.analyze(f)
				}
			case *ast.AssignStmt:
				for _, r := range d.Rhs {
					f, ok := r.(*ast.CallExpr)
					if ok {
						g.analyze(f)
					}
				}
			}
		}
	*/
}

func (g *gomock) extractInterfaceFunc(pkg *packages.Package, asts []*ast.File, funcName string) {
	for _, a := range asts {
		for _, decl := range a.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				g.walkFuncBody(pkg, d.Body)
			}
		}
	}
}

func (g *gomock) walkFuncBody(pkg *packages.Package, bs *ast.BlockStmt) {
	for _, stmt := range bs.List {
		switch d := stmt.(type) {
		case *ast.AssignStmt:
			// fmt.Println(d)
		case *ast.ExprStmt:
			callExpr, ok := d.X.(*ast.CallExpr)
			fmt.Println(callExpr)
			if ok {
				fmt.Println(pkg.TypesInfo.Types[callExpr].Type.String())
			}
		}
	}
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

func (g *gomock) analyze(callExpr *ast.CallExpr) error {
	for _, pkg := range g.pkgs {
		f, ok := pkg.TypesInfo.Types[callExpr]
		if !ok {
			continue
		}
		fmt.Println(f)
	}
	return nil
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
