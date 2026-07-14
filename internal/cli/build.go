package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/notblankz/forge/internal/site"
)

func Build(args []string) error {
	opts, err := parseBuildOptions(args)
	if err != nil {
		return err
	}

	return site.Build(opts)
}

func parseBuildOptions(args []string) (site.BuildOptions, error) {
	if len(args) == 0 {
		return site.BuildOptions{}, errors.New("build: content directory required")
	}

	cur := site.BuildOptions{}

	// Check for existence of Input root directory
	root := args[0]
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return site.BuildOptions{}, fmt.Errorf("build: content directory %q does not exist", root)
		}
		return site.BuildOptions{}, fmt.Errorf("build: cannot access content directory %q: %w", root, err)
	}
	cur.ContentRoot = root
	cur.DestRoot = "dist"

	return cur, nil
}
