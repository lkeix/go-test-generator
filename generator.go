package gotestgenerator

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	estimatenecessarytests "github.com/lkeix/estimate-necessary-tests"
)

type Generator struct {
	funcs map[string]map[string]int64
}

func NewGenerator(path string) (*Generator, error) {
	astLoader := estimatenecessarytests.NewASTLoader(path, false)
	err := astLoader.Load()
	if err != nil {
		return nil, err
	}

	funcs := make(map[string]map[string]int64)
	for k, ast := range astLoader.Asts {
		calculator := estimatenecessarytests.NewCalculator()
		calculator.Calculate(ast)
		funcs[k] = calculator.Result
	}
	return &Generator{
		funcs: funcs,
	}, nil
}

func (g *Generator) Generate() {
	// TODO: generate each funcs test code into Xxx_test.go
	for path, funcs := range g.funcs {
		dname := filepath.Dir(path)
		bname := filepath.Base(path)
		testFileName := buildTestFileName(bname)

		output := fmt.Sprintf("%s/%s", dname, testFileName)
		pairs := make(map[string]int64)
		var packageName string
		for fInfo, f := range funcs {
			split := strings.Split(fInfo, ".")
			pName, funcName, isPublic := buildTestCaseFuncName(split)
			if isPublic {
				pairs[funcName] = f
			}
			packageName = pName
		}

		f := BuildTestCase(packageName, pairs)

		if packageName == "" {
			continue
		}

		if _, err := os.Stat(output); os.IsNotExist(err) {
			file, err := os.Create(output)
			if err != nil {
				log.Fatal(err)
			}
			err = format.Node(file, token.NewFileSet(), f)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func buildTestFileName(name string) string {
	split := strings.Split(name, ".")
	return split[0] + "_test." + split[1]
}

func buildTestCaseFuncName(split []string) (string, string, bool) {
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
		builder.Write([]byte(structName))
		builder.Write([]byte(funcName))
		return fmt.Sprintf("%s_test", packageName), builder.String(), true
	}
	return "", "", false
}

func BuildTestCase(packageName string, funcs map[string]int64) *ast.File {
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

	decls := buildTestFuncDecls(funcs)
	f.Decls = append(f.Decls, decls...)

	return f
}

func buildTestFuncDecls(funcs map[string]int64) []ast.Decl {
	decls := []ast.Decl{}
	for testFuncName, needTestCases := range funcs {
		fmt.Println(testFuncName)
		fmt.Println(needTestCases)
		decls = append(decls, &ast.FuncDecl{
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
			Body: buildTestcase(needTestCases),
		})
	}

	return decls
}

func buildTestcase(num int64) *ast.BlockStmt {
	buildTestcaseStruct := func(num int64) []ast.Expr {
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
			&ast.RangeStmt{
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
			},
		},
	}
}
