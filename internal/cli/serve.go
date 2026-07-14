package cli

import (
	"flag"

	"github.com/notblankz/forge/internal/serve"
)

func Serve(args []string) error {
	opts, err := parseServeOptions(args)
	if err != nil {
		return err
	}
	return serve.Serve(opts)
}

func parseServeOptions(args []string) (serve.ServeOptions, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := fs.Int("port", 3000, "port to serve on")
	if err := fs.Parse(args); err != nil {
		return serve.ServeOptions{}, err
	}

	// remaining positional arg = content dir, reuse build's validation
	buildOpts, err := parseBuildOptions(fs.Args())
	if err != nil {
		return serve.ServeOptions{}, err
	}

	return serve.ServeOptions{BuildOptions: buildOpts, Port: *port}, nil
}
