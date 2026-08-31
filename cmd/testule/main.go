package main

import (
	"os"

	"github.com/hackelia-micrantha/testule/internal/cli"
)

func main() {
	os.Exit(cli.RunWithInput(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
