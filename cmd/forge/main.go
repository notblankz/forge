package main

import (
	"fmt"
	"os"

	"github.com/notblankz/forge/internal/cli"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Help")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "build":
		if err := cli.Build(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	case "serve":
		if err := cli.Serve(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "bench":
		cli.Bench(os.Args[2:])
	default:
		fmt.Println("Help")
		os.Exit(2)
	}
}
