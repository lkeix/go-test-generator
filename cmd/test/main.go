package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
)

func main() {
	src := `// Package main is the location of the entry point for the program
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	// Create the AST by parsing src.
	fiset := token.NewFileSet() // positions are relative to the file set
	file, err := parser.ParseFile(fiset, "", src, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// Check if there are comments in the AST.
	if len(file.Comments) > 0 {
		// Assuming the first comment is at the file's head.
		headComment := file.Comments[0]
		for _, comment := range headComment.List {
			fmt.Println(comment.Text)
		}
	} else {
		fmt.Println("No comments found.")
	}
}
