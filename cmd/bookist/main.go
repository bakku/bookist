package main

import (
	"os"

	"bakku.dev/bookist/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
