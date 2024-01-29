package gomock

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GomockComment string

func NewGomockCommnet(comment string) GomockComment {
	return GomockComment(comment)
}

const dstOptionString = "-destination="

func (g GomockComment) HasMockComment() bool {
	return g != ""
}

func (g GomockComment) DstPath() string {
	splits := strings.Split(string(g), " ")

	for _, s := range splits {
		if strings.HasPrefix(s, dstOptionString) {
			relPath := strings.ReplaceAll(s, dstOptionString, "")
			absPath, _ := filepath.Abs(relPath)
			return absPath
		}
	}

	return ""
}

const srcOptionString = "-source="

func (g GomockComment) SrcPath() string {
	splits := strings.Split(string(g), " ")

	for _, s := range splits {
		if strings.HasPrefix(s, srcOptionString) {
			relPath := strings.ReplaceAll(s, srcOptionString, "")
			absPath, _ := filepath.Abs(relPath)
			return absPath
		}
	}

	return ""
}

type Gomock interface {
	ExtractDepsInterface(filePath, funcName string) map[*ast.CallExpr]struct{}
	BuildMockSkelton(callExpr *ast.CallExpr) []ast.Expr
	ExtractGoMockComment(filepath string) (GomockComment, error)
	ExtractMockPkgPath(path string) map[string]*packages.Package
	IsImportedMockPkg(filePath string, mockPkg string) bool
}

type FilePath string

type FuncName string

type gomock struct {
	InterfaceImportMap map[string]string
	pkgs               []*packages.Package
	funcDecls          map[FilePath]map[FuncName]*ast.FuncDecl
	asts               map[string]*ast.File
}

func (g *gomock) ExtractGoMockComment(filepath string) (GomockComment, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}
	if len(f.Comments) > 0 {
		return GomockComment(f.Comments[0].Text()), nil
	}

	return "", nil
}

func (g *gomock) BuildInterfaceImportMap(filePath string) {

}

func (g *gomock) IsImportedMockPkg(filePath string, mockPkg string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return false
	}

	for _, im := range f.Imports {
		if im.Name == nil {
			continue
		}
		if im.Name.Name == mockPkg {
			return true
		}
	}

	return false
}

func (g *gomock) BuildMockSkelton(callExpr *ast.CallExpr) []ast.Expr {
	return nil
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

func (g *gomock) ExtractMockPkgPath(path string) map[string]*packages.Package {
	for _, pkg := range g.pkgs {
		if slices.Contains(pkg.GoFiles, path) {
			return pkg.Imports
		}
	}

	return nil
}

func (g *gomock) extractInterfaceFunc(pkg *packages.Package, asts []*ast.File, funcName string) *ast.CallExpr {
	for _, a := range asts {
		for _, decl := range a.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.Name != funcName {
					continue
				}
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
						fmt.Println(recv.Pkg().Name())
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
					fmt.Println(recv.Pkg().Name())
					return callExpr
				}
			}
		case *ast.IfStmt:
			if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
				for _, expr := range assignStmt.Rhs {
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
							fmt.Println(recv.Pkg().Name())
							return callExpr
						}
					}
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
