package cmd

import (
	"github.com/notblankz/forge/internal/serve"
	"github.com/notblankz/forge/internal/site"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve <content-dir>",
	Short: "Start the dev server with live rebuild",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := serve.Config{
			BuildOptions: site.BuildOptions{
				ContentRoot: args[0],
				DestRoot:    "dist",
			},
			Port: servePort,
		}
		return serve.Start(opts)
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 3000, "port to serve on")
	rootCmd.AddCommand(serveCmd)
}
