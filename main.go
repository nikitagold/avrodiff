package main

import (
	"os"

	"github.com/nikitagold/avrodiff/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
