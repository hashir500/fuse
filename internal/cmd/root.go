package cmd

import (
	"fmt"
	"os"

	"github.com/hashir500/Fuse/internal/spark"
	"github.com/spf13/cobra"
)

var quiet bool
var noMascot bool

var rootCmd = &cobra.Command{
	Use:   "fuse",
	Short: "Set a fuse on your AI spending",
	Long:  "Fuse is a lightweight local proxy that tracks AI API spend and enforces hard budget caps.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		spark.SetQuiet(quiet)
		spark.SetNoMascot(noMascot)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		spark.ConfigInvalid(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress Spark mascot output")
	rootCmd.PersistentFlags().BoolVar(&noMascot, "no-mascot", false, "hide Spark ASCII art but keep inline messages")
}
