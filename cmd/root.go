package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "This is a plugin that helps to visualize the different components within a Spin application",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var All bool

func init() {
	showCmd.PersistentFlags().StringP("file", "f", "", "Specifies the path to the spin.toml file you wish to visualize")
	showCmd.PersistentFlags().StringP("env", "e", "", "Specifies the path to the \".env\" file containing your Spin variables")
    showCmd.PersistentFlags().BoolVarP(&All, "all", "a", false, "Output information about all component. Only applies if no component name is specified.")
	rootCmd.AddCommand(showCmd)
}
