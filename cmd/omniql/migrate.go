package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

// runMigrateMode handles the 'migrate' subcommand.
func runMigrateMode() {
	configFlag := flag.String("config", "omniql.yaml", "Path to omniql.yaml config file")
	sourceFlag := flag.String("source", "", "Source target name")
	destFlag   := flag.String("dest", "", "Destination target name")
	batchFlag  := flag.Int("batch", 100, "Batch size")
	dryRunFlag := flag.Bool("dry-run", false, "Dry run (don't write)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: omniql migrate -config <file> -source <target> -dest <target> [options]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *sourceFlag == "" || *destFlag == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg == nil {
		log.Fatalf("config file not found at %q", *configFlag)
	}

	engine := core.NewEngine()
	if err := applyConfig(engine, cfg); err != nil {
		log.Fatal(err)
	}

	opts := core.MigrateOptions{
		BatchSize: *batchFlag,
		DryRun:    *dryRunFlag,
	}

	fmt.Fprintf(os.Stderr, "Migrating from %q to %q...\n", *sourceFlag, *destFlag)
	result, err := engine.Migrate(context.Background(), *sourceFlag, *destFlag, opts)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}
}
