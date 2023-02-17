package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	flagProjectDir       string
	flagReset            bool
	flagPort             string
	flagNodePackagesPath string
	flagPattern          string
)

var rootCmd = &cobra.Command{
	Use:   "keel",
	Short: "The Keel CLI",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().StringVarP(&flagProjectDir, "dir", "d", workingDir, "directory containing a Keel project")
}
