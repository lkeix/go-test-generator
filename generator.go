package gotestgenerator

import estimatenecessarytests "github.com/lkeix/estimate-necessary-tests"

type Generator struct {
	funcs map[string]int64
}

func NewGenerator(path string) (*Generator, error) {
	astLoader := estimatenecessarytests.NewASTLoader(path, false)
	err := astLoader.Load()
	if err != nil {
		return nil, err
	}

	calculator := estimatenecessarytests.NewCalculator()
	for _, ast := range astLoader.Asts {
		calculator.Calculate(ast)
	}
	return &Generator{
		funcs: calculator.Result,
	}, nil
}

func (g *Generator) Generate() {
	// TODO: generate each funcs test code into Xxx_test.go
}
