package cmd

import (
	"github.com/spf13/cobra"
	"github.com/teamkeel/keel/cmd/program"
	"github.com/teamkeel/keel/util"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests",
	Run: func(cmd *cobra.Command, args []string) {
		port, err := util.GetFreePort()
		if err != nil {
			panic(err)
		}

		program.Run(&program.Model{
			Mode:             program.ModeTest,
			ProjectDir:       flagProjectDir,
			Port:             port,
			NodePackagesPath: flagNodePackagesPath,
			PrivateKeyPath:   flagPrivateKeyPath,
			TestPattern:      flagPattern,
		})
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&flagPattern, "pattern", "p", "(.*)", "pattern to isolate test")
	testCmd.Flags().StringVar(&flagPrivateKeyPath, "private-key-path", "", "path to the private key .pem file")

	if Debug {
		testCmd.Flags().StringVar(&flagNodePackagesPath, "node-packages-path", "", "path to local @teamkeel npm packages")
	}
}
