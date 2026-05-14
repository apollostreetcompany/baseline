package main

import (
	"os"

	"github.com/apollostreetcompany/baseline/internal/baseline"
)

func main() {
	os.Exit(baseline.Main(os.Args[1:], os.Stdout, os.Stderr))
}
