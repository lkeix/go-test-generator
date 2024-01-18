//go:generate mockgen -source=$GOFILE -package=usecase -destination=../test/mock/usecase/admin.go
package gotestgenerator

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	estimatenecessarytests "github.com/lkeix/estimate-necessary-tests"
	"github.com/lkeix/go-test-generator/gomock"
)

type Generator struct {
	enableGoMock bool
	astLoader    *estimatenecessarytests.ASTLoader
	gm           gomock.Gomock
}

func NewGenerator(path string, enableGoMock bool) (*Generator, error) {
	astLoader := estimatenecessarytests.NewASTLoader(path, false)
	err := astLoader.Load(parser.ParseComments)
	if err != nil {
		return nil, err
	}

	gm, err := gomock.NewGomock(path, astLoader.Asts)
	if err != nil {
		return nil, err
	}

	return &Generator{
		astLoader:    astLoader,
		enableGoMock: enableGoMock,
		gm:           gm,
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
		var testPackageName string

		outputDecls := make([]ast.Decl, 0)
		for fInfo, f := range funcs {
			split := strings.Split(fInfo, ".")

			funcName := extractTestTargetFuncName(split)
			isExported := IsExportedFunc(funcName)
			testPackageName = createTestPackageName(split)
			testFuncName := createTestCaseFuncName(split)

			if isExported {
				abs, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				decl := g.BuildTestCase(fsets[path], abs, funcName, testFuncName, int(f))
				outputDecls = append(outputDecls, decl)
			}
		}

		if testPackageName == "" {
			continue
		}

		outputASTFile := newTestCodeASTFile(testPackageName)
		outputASTFile.Decls = append(outputASTFile.Decls, outputDecls...)
		if len(outputDecls) == 0 {
			continue
		}

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

func createTestPackageName(split []string) string {
	packageName := split[0]
	return fmt.Sprintf("%s_test", packageName)
}

func extractTestTargetFuncName(split []string) string {
	if len(split) == 2 {
		return split[1]
	}

	if len(split) == 3 {
		return split[2]
	}

	return ""
}

func IsExportedFunc(funcName string) bool {
	if len(funcName) == 0 {
		return false
	}
	return unicode.IsUpper(rune(funcName[0]))
}

func createTestCaseFuncName(split []string) string {
	var structName, funcName string
	if len(split) == 2 {
		funcName = split[1]
	}

	if len(split) == 3 {
		structName = split[1]
		funcName = split[2]
	}

	var builder strings.Builder
	builder.Write([]byte("Test"))
	if structName != "" {
		builder.Write([]byte(structName))
		builder.Write([]byte("_"))
	}

	builder.Write([]byte(funcName))
	return builder.String()
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

func (g *Generator) BuildTestCase(f *ast.File, testTargetFilePath, testTargetFuncName, testFuncName string, needsTestCasesNumber int) ast.Decl {
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
		Body: g.buildSkeltonTestCode(f, testTargetFilePath, testTargetFuncName, needsTestCasesNumber),
	}
	return decl
}

func (g *Generator) buildSkeltonTestCode(f *ast.File, testTargetFilePath, testTargetFuncName string, num int) *ast.BlockStmt {
	interfaceFunc := g.gm.ExtractDepsInterface(testTargetFilePath, testTargetFuncName)
	for callExpr := range interfaceFunc {
		fmt.Println(callExpr)
	}

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
