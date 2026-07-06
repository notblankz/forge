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
		cli.Build(os.Args[2:])
	case "serve":
		cli.Serve(os.Args[2:])
	case "bench":
		cli.Bench(os.Args[2:])
	default:
		fmt.Println("Help")
		os.Exit(2)
	}
}
