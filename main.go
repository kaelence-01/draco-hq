// Command draco documents deploys from the terminal.
package main

import (
	"os"

	"github.com/kaelence-01/draco-hq/cmd"
)

func main() {
	os.Exit(cmd.New(os.Stdin, os.Stdout, os.Stderr).Start(os.Args[1:]))
}
