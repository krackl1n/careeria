package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/krackl1n/careeria/contracts/c4"
)

func main() {
	configPath := flag.String("config", "docs/architecture/c4-l4.yaml", "path to c4-l4 config")
	flag.Parse()

	if err := c4.GenerateFromConfig(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "c4gen error: %v\n", err)
		os.Exit(1)
	}
}
