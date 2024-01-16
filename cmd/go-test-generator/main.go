package main

import (
	"log"

	"github.com/lkeix/go-test-generator/cli"
)

func main() {
	c := cli.NewCLI()
	if err := c.Execute(); err != nil {
		log.Fatal(err)
	}
}
