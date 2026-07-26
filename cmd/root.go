package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "forge is a fast static site generator",
	Long:  "forge is a fast, concurrent static site generator for personal websites.",
}

var timing bool

// Execute runs the root command and handles errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&timing, "timing", false, "print per-phase build timings")
}
