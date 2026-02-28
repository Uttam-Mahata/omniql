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
	sourceFlag := flag.String("source", "", "Source target name (single table)")
	destFlag   := flag.String("dest", "", "Destination target name (single table)")
	sourceDriverFlag := flag.String("source-driver", "", "Source driver name (full migration)")
	destDriverFlag   := flag.String("dest-driver", "", "Destination driver name (full migration)")
	batchFlag  := flag.Int("batch", 100, "Batch size")
	dryRunFlag := flag.Bool("dry-run", false, "Dry run (don't write)")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  Single table: omniql migrate -config <file> -source <target> -dest <target> [options]")
		fmt.Fprintln(os.Stderr, "  Full DB:      omniql migrate -config <file> -source-driver <name> -dest-driver <name> [options]")
		flag.PrintDefaults()
	}
	flag.Parse()

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

	var result *core.MigrateResult

	if *sourceFlag != "" && *destFlag != "" {
		// Single table migration
		fmt.Fprintf(os.Stderr, "Migrating target %q to %q...\n", *sourceFlag, *destFlag)
		result, err = engine.Migrate(context.Background(), *sourceFlag, *destFlag, opts)
	} else if *sourceDriverFlag != "" && *destDriverFlag != "" {
		// Full database migration
		fmt.Fprintf(os.Stderr, "Migrating all targets from driver %q to %q...\n", *sourceDriverFlag, *destDriverFlag)
		result, err = engine.MigrateAll(context.Background(), *sourceDriverFlag, *destDriverFlag, opts)
	} else {
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}
}
