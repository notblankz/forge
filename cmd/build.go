package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/notblankz/forge/internal/site"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build <site-dir>",
	Short: "Build the site into the output directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		siteRoot := args[0]
		if _, err := os.Stat(siteRoot); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("build: site directory %q does not exist", siteRoot)
			}
			return fmt.Errorf("build: cannot access site directory %q: %w", siteRoot, err)
		}
		return site.Build(site.BuildOptions{SiteRoot: siteRoot})
	},
}

func init() {
	// TODO: also add --output flag 
	rootCmd.AddCommand(buildCmd)
}
