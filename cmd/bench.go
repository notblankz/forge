package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark the build",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("bench: (not implemented)\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(benchCmd)
}
