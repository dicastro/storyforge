// Package cmd contains all storyforge CLI commands built with cobra.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var contentRoot string

var rootCmd = &cobra.Command{
	Use:   "storyforge",
	Short: "A CLI for managing and publishing picture books",
	Long: `Storyforge helps you organise, validate and generate publication
materials for picture books sold through Amazon KDP and other distributors.

Content lives in YAML files on disk — no database required.
Run 'storyforge help <command>' for detailed usage.`,
}

// Execute adds all child commands to the root command and sets flags.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&contentRoot,
		"content",
		"",
		"path to content directory (overrides STORYFORGE_CONTENT env var)",
	)

	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newShowCmd())
}