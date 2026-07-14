package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/notblankz/forge/internal/site"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build <content-dir>",
	Short: "Build the site into the output directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contentRoot := args[0]
		if _, err := os.Stat(contentRoot); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("build: content directory %q does not exist", contentRoot)
			}
			return fmt.Errorf("build: cannot access content directory %q: %w", contentRoot, err)
		}
		opts := site.BuildOptions{
			ContentRoot: args[0],
			DestRoot:    "dist",
		}
		return site.Build(opts)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
