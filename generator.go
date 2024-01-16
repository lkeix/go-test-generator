package gotestgenerator

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	estimatenecessarytests "github.com/lkeix/estimate-necessary-tests"
)

type Generator struct {
	enableGoMock bool
	astLoader    *estimatenecessarytests.ASTLoader
}

func NewGenerator(path string, enableGoMock bool) (*Generator, error) {
	astLoader := estimatenecessarytests.NewASTLoader(path, false)
	err := astLoader.Load()
	if err != nil {
		return nil, err
	}

	return &Generator{
		astLoader:    astLoader,
		enableGoMock: enableGoMock,
	}, nil
}

func (g *Generator) Generate() error {
	funcsMap := make(map[string]map[string]int64)
	fsets := make(map[string]*ast.File)
	for k, ast := range g.astLoader.Asts {
		calculator := estimatenecessarytests.NewCalculator()
		calculator.Calculate(ast)
		funcsMap[k] = calculator.Result
		fsets[k] = ast
	}

	for path, funcs := range funcsMap {
		dname := filepath.Dir(path)
		bname := filepath.Base(path)
		testFileName := createTestFileName(bname)

		output := fmt.Sprintf("%s/%s", dname, testFileName)
		var packageName string

		outputDecls := make([]ast.Decl, 0)
		for fInfo, f := range funcs {
			split := strings.Split(fInfo, ".")
			pName, funcName, isPublic := createTestCaseFuncName(split)
			packageName = pName

			if isPublic {
				fmt.Println(fsets[path].Comments)
				decl := g.BuildTestCase(fsets[path], funcName, int(f))
				outputDecls = append(outputDecls, decl)
			}
		}

		if packageName == "" {
			continue
		}

		outputASTFile := newTestCodeASTFile(packageName)
		outputASTFile.Decls = append(outputASTFile.Decls, outputDecls...)
		if _, err := os.Stat(output); os.IsNotExist(err) {
			file, err := os.Create(output)
			if err != nil {
				return err
			}
			err = format.Node(file, token.NewFileSet(), outputASTFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createTestFileName(name string) string {
	split := strings.Split(name, ".")
	return split[0] + "_test." + split[1]
}

func createTestCaseFuncName(split []string) (string, string, bool) {
	var packageName, structName, funcName string
	packageName = split[0]
	if len(split) == 2 {
		funcName = split[1]
	}

	if len(split) == 3 {
		structName = split[1]
		funcName = split[2]
	}

	var builder strings.Builder
	builder.Write([]byte("Test"))
	if unicode.IsUpper(rune(funcName[0])) {
		if structName != "" {
			builder.Write([]byte(structName))
			builder.Write([]byte("_"))
		}
		builder.Write([]byte(funcName))
		return fmt.Sprintf("%s_test", packageName), builder.String(), true
	}
	return "", "", false
}

func newTestCodeASTFile(packageName string) *ast.File {
	f := &ast.File{
		Name: ast.NewIdent(packageName),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: strconv.Quote("fmt"),
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: strconv.Quote("testing"),
						},
					},
				},
			},
		},
	}
	return f
}

func (g *Generator) BuildTestCase(f *ast.File, testFuncName string, needsTestCasesNumber int) ast.Decl {
	decl := &ast.FuncDecl{
		Name: ast.NewIdent(testFuncName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							{
								Name: "t",
							},
						},
						Type: &ast.StarExpr{
							X: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "testing"},
								Sel: &ast.Ident{Name: "T"},
							},
						},
					},
				},
			},
		},
		Body: g.buildSkeltonTestCode(f, needsTestCasesNumber),
	}
	return decl
}

func (g *Generator) buildSkeltonTestCode(f *ast.File, num int) *ast.BlockStmt {
	buildTestcaseStruct := func(num int) []ast.Expr {
		var exprs []ast.Expr
		compositLit := &ast.CompositeLit{
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key: &ast.Ident{
						Name: "name",
					},
					Value: &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(""),
					},
				},
			},
		}

		for i := 0; i < int(num); i++ {
			exprs = append(exprs, compositLit)
		}

		return exprs
	}

	for _, comment := range f.Comments {
		fmt.Println(comment.List[0].Text)
	}

	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{
						Name: "testcases",
					},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.ArrayType{
							Elt: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: []*ast.Ident{
												{
													Name: "name",
												},
											},
											Type: &ast.Ident{
												Name: "string",
											},
										},
									},
								},
							},
						},
						Elts: buildTestcaseStruct(num),
					},
				},
			},
			buildRunTestForStmt(),
		},
	}
}

func buildRunTestForStmt() *ast.RangeStmt {
	return &ast.RangeStmt{
		Key: &ast.Ident{
			Name: "_",
		},
		Value: &ast.Ident{
			Name: "testcase",
		},
		Tok: token.DEFINE,
		X: &ast.Ident{
			Name: "testcases",
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "t",
							},
							Sel: &ast.Ident{
								Name: "Run",
							},
						},
						Args: []ast.Expr{
							&ast.SelectorExpr{
								X: &ast.Ident{
									Name: "testcase",
								},
								Sel: &ast.Ident{
									Name: "name",
								},
							},
							&ast.FuncLit{
								Type: &ast.FuncType{
									Params: &ast.FieldList{
										List: []*ast.Field{
											{
												Names: []*ast.Ident{
													{
														Name: "t",
													},
												},
												Type: &ast.StarExpr{
													X: &ast.SelectorExpr{
														X:   &ast.Ident{Name: "testing"},
														Sel: &ast.Ident{Name: "T"},
													},
												},
											},
										},
									},
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X: &ast.Ident{
														Name: "fmt",
													},
													Sel: &ast.Ident{
														Name: "Println",
													},
												},
												Args: []ast.Expr{
													&ast.BasicLit{
														Kind:  token.STRING,
														Value: "\"write your unit test!\"",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
