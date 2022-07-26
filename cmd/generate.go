package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/teamkeel/keel/functions/runtime"
)

var NODE_MODULE_DIR string = ".keel"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates code",
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaDir, _ := cmd.Flags().GetString("dir")

		r, err := runtime.NewRuntime(schemaDir, filepath.Join(schemaDir, "node_modules", NODE_MODULE_DIR))

		if err != nil {
			return err
		}

		_, err = r.Generate()

		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	defaultDir, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	generateCmd.Flags().StringVarP(&inputDir, "dir", "d", defaultDir, "the directory containing the Keel schema files")
}
