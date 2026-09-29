package main

import (
	"fmt"
	"io"
	"os"

	"sitewise/internal/config"
	"sitewise/internal/latency"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr, os.Stdout))
}

func run(args []string, getenv func(string) string, stderr, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "gate" {
		return latency.RunGate(args[1:], stderr)
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	fmt.Fprintf(stdout, "sitewise ready model=%s\n", cfg.JevModel)
	return 0
}
