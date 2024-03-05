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

func (g GomockComment) ExtractMockSrcPath() MockSrcPath {
	lines := strings.Split(string(g), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Source: ") {
			rel := strings.ReplaceAll(line, "Source: ", "")
			absPath, _ := filepath.Abs(rel)
			return MockSrcPath(absPath)
		}
	}
	return ""
}

func (g GomockComment) ExtractGeneratedMockPath() GeneratedMockPath {
	lines := strings.Split(string(g), "\n")
	for _, line := range lines {
		splits := strings.Split(line, " ")
		for _, s := range splits {
			if strings.HasPrefix(s, dstOptionString) {
				relPath := strings.ReplaceAll(s, dstOptionString, "")
				absPath, _ := filepath.Abs(relPath)
				return GeneratedMockPath(absPath)
			}
		}
	}
	return ""
}

type MockSrcPath string

type GeneratedMockPath string

type MockInfo struct {
	GeneratedMockPath   GeneratedMockPath
	DeclearedInterfaces []*Interface
}

type Interface struct {
	Name string
	I    *ast.InterfaceType
}

type MockMap map[MockSrcPath]*MockInfo

func NewMockMap() MockMap {
	return make(MockMap)
}

func ExtractDeclearedInterfaces(decls []ast.Decl) []*Interface {
	is := make([]*Interface, 0)
	for _, decl := range decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			name, i := extractInterface(d.Specs)
			if name == "" && i == nil {
				continue
			}
			is = append(is, &Interface{
				Name: name,
				I:    i,
			})
		}
	}
	return is
}

func extractInterface(specs []ast.Spec) (string, *ast.InterfaceType) {
	for _, spec := range specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}

		return typeSpec.Name.Name, interfaceType
	}

	return "", nil
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
	IsReferedFrom(m MockMap, fset *ast.File, path string) bool
	AST() map[string]*ast.File
}

type FilePath string

type FuncName string

type gomock struct {
	InterfaceImportMap map[string]string
	pkgs               []*packages.Package
	funcDecls          map[FilePath]map[FuncName]*ast.FuncDecl
	asts               map[string]*ast.File
	module             string
}

type InterfaceDepsDirection struct {
	Caller   *ast.FuncDecl
	CallFunc *types.Interface
}

func (g *gomock) IsReferedFrom(m MockMap, fset *ast.File, path string) bool {
	decls := fset.Decls
	for _, d := range decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}

		for _, stmt := range f.Body.List {
			switch s := stmt.(type) {
			case *ast.AssignStmt:
				for _, expr := range s.Rhs {
					if callExpr, ok := expr.(*ast.CallExpr); ok {
						funcObj := g.extractReference(g.pkgs, callExpr)
						if funcObj == nil {
							continue
						}

						recv := extractReciver(funcObj)
						if recv == nil {
							continue
						}

						if _, ok := recv.Type().Underlying().(*types.Interface); ok {
							return true
						}
					}
				}
			case *ast.ExprStmt:
				callExpr, ok := s.X.(*ast.CallExpr)
				if ok {
					funcObj := g.extractReference(g.pkgs, callExpr)
					if funcObj == nil {
						continue
					}

					recv := extractReciver(funcObj)
					if recv == nil {
						continue
					}

					if _, ok := recv.Type().Underlying().(*types.Interface); ok {
						return true
					}
				}
			case *ast.IfStmt:
				if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
					for _, expr := range assignStmt.Rhs {
						if callExpr, ok := expr.(*ast.CallExpr); ok {
							funcObj := g.extractReference(g.pkgs, callExpr)
							if funcObj == nil {
								continue
							}

							recv := extractReciver(funcObj)
							if recv == nil {
								continue
							}
							fmt.Println(funcObj.Name())

							if _, ok := recv.Type().Underlying().(*types.Interface); ok {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
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
					funcObj := g.extractReference(g.pkgs, callExpr)
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
				funcObj := g.extractReference(g.pkgs, callExpr)
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
		case *ast.IfStmt:
			if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
				for _, expr := range assignStmt.Rhs {
					if callExpr, ok := expr.(*ast.CallExpr); ok {
						funcObj := g.extractReference(g.pkgs, callExpr)
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
		}
	}
	return nil
}

func (g *gomock) AST() map[string]*ast.File {
	res := make(map[string]*ast.File)
	for _, pkg := range g.pkgs {
		for i, file := range pkg.GoFiles {
			res[file] = pkg.Syntax[i]
		}
	}
	return res
}

func (gm *gomock) extractReference(pkgs []*packages.Package, callExpr *ast.CallExpr) types.Object {
	for _, pkg := range pkgs {
		if gm.module != "" && strings.HasPrefix(pkg.String(), gm.module) {
			switch f := callExpr.Fun.(type) {
			case *ast.Ident:
				if res := pkg.TypesInfo.ObjectOf(f); res != nil {
					if gm.module == "" {
						return res
					}
					if res.Pkg() != nil {
						if gm.module != "" && strings.Contains(res.Pkg().String(), gm.module) {
							return res
						}
					}
				}
			case *ast.SelectorExpr:
				if res := pkg.TypesInfo.ObjectOf(f.Sel); res != nil {
					if gm.module == "" {
						return res
					}

					if res.Pkg() != nil {
						if gm.module != "" && strings.Contains(res.Pkg().String(), gm.module) {
							return res
						}
					}
				}
			}
		}
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

func NewGomock(rootDir string, asts map[string]*ast.File, module string) (Gomock, error) {
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
		module:    module,
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
