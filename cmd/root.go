// Package cmd contains all Cobra command definitions.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "sugo",
	Short: "SuggestGreatOutput — GitHub PR review assistant",
	Long:  `sugo fetches a GitHub PR, runs analysis agents in parallel, and produces a structured review report.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".sugo.yaml", "config file")
}
