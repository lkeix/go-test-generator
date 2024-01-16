package cli

import (
	"log"

	gotestgenerator "github.com/lkeix/go-test-generator"
	"github.com/spf13/cobra"
)

func NewCLI() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate-test-skeleton",
		Short: "generate skeleton code",
		Run: func(cmd *cobra.Command, args []string) {
			path, err := cmd.Flags().GetString("path")
			if err != nil {
				log.Fatal(err)
			}

			enableGoMock, err := cmd.Flags().GetBool("enable-go-mock")
			if err != nil {
				log.Fatal(err)
			}

			generator, err := gotestgenerator.NewGenerator(path, enableGoMock)
			if err != nil {
				log.Fatal(err)
			}

			if err := generator.Generate(); err != nil {
				log.Fatal(err)
			}
		},
	}

	var path string
	c.Flags().StringVar(&path, "path", "", "specify gererate go test path")

	var enableGomock bool
	c.Flags().BoolVar(&enableGomock, "enable-go-mock", false, "specify enable go mock")

	return c
}
