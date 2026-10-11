package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"synora/internal/intelligence/datasetengine"
)

func main() {
	seed := flag.Int64("seed", 1, "deterministic generator seed")
	homes := flag.Int("homes", datasetengine.DefaultHomeCount, "number of synthetic homes (6–256)")
	start := flag.String("start-time", "2026-01-01T00:00:00Z", "synthetic start time in RFC3339 format")
	output := flag.String("output", "", "new or dedicated output directory for separated artifacts")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "--output is required")
		os.Exit(2)
	}
	startTime, err := time.Parse(time.RFC3339, *start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --start-time: %v\n", err)
		os.Exit(2)
	}
	dataset, err := datasetengine.Generate(datasetengine.Config{Seed: *seed, HomeCount: *homes, StartTime: startTime})
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate dataset: %v\n", err)
		os.Exit(1)
	}
	if err := dataset.WriteArtifacts(*output); err != nil {
		fmt.Fprintf(os.Stderr, "write dataset artifacts: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("schema=%s homes=%d examples=%d output=%s\n", dataset.Manifest.SchemaVersion, dataset.Manifest.HomeCount, dataset.Manifest.ExampleCount, *output)
}
