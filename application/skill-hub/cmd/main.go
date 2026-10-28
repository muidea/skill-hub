package main

import (
	"fmt"
	"os"

	cli "github.com/muidea/skill-hub/internal/clis/skill-hub"
)

func main() {
	if err := cli.Execute(); err != nil {
		if !cli.ErrorReported(err) {
			fmt.Fprintf(os.Stderr, "Error: %s\n", cli.FormatError(err))
		}
		os.Exit(1)
	}
}
