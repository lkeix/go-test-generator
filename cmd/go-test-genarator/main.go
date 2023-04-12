package main

import (
	"log"

	testgenerator "github.com/lkeix/go-test-generator"
)

func main() {
	g, err := testgenerator.NewGenerator(".")
	if err != nil {
		log.Fatal(err)
	}

	g.Generate()
}
