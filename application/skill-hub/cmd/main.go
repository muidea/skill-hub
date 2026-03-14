package main

import (
	"fmt"
	"os"

	hubmodule "github.com/muidea/skill-hub/internal/modules/kernel/hub"
)

func main() {
	if err := hubmodule.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
