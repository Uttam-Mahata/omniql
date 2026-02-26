// Package main is the OmniQL CLI entry-point.
//
// It supports two modes of operation:
//
//  1. Flag mode: specify -driver, -dsn, and -query directly.
//  2. Config mode: load an omniql.yaml file (via -config or the default
//     omniql.yaml in the current directory) and specify only -query.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvels "github.com/Uttam-Mahata/omniql/pkg/drivers/elasticsearch"
	drvmongo "github.com/Uttam-Mahata/omniql/pkg/drivers/mongo"
	drvmysql "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"
	drvpostgres "github.com/Uttam-Mahata/omniql/pkg/drivers/postgres"
	drvredis "github.com/Uttam-Mahata/omniql/pkg/drivers/redis"
	drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
	drvsqlserver "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver"
	_ "github.com/go-sql-driver/mysql"  // MySQL database/sql driver
	_ "github.com/lib/pq"               // Postgres database/sql driver
	_ "github.com/microsoft/go-mssqldb" // SQL Server database/sql driver
	"go.mongodb.org/mongo-driver/mongo"
	mongoopts "go.mongodb.org/mongo-driver/mongo/options"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: omniql [options] -query '<OQL JSON>'")
	fmt.Fprintln(os.Stderr, "       omniql shell [-config omniql.yaml]")
	fmt.Fprintln(os.Stderr, "       omniql migrate -config omniql.yaml -source <target> -dest <target> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flag mode (single driver):")
	fmt.Fprintln(os.Stderr, "  -driver   Database driver: sqlite, postgres, mongo, mysql, sqlserver, redis, or elasticsearch")
	fmt.Fprintln(os.Stderr, "  -dsn      Connection string / file path / MongoDB URI / Redis URL / Elasticsearch address")
	fmt.Fprintln(os.Stderr, "  -db       Database name (required for mongo)")
	fmt.Fprintln(os.Stderr, "  -query    JSON-encoded OQL query to execute")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Config mode (omniql.yaml):")
	fmt.Fprintln(os.Stderr, "  -config   Path to omniql.yaml (default: ./omniql.yaml)")
	fmt.Fprintln(os.Stderr, "  -query    JSON-encoded OQL query to execute")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Shell mode (interactive REPL):")
	fmt.Fprintln(os.Stderr, "  omniql shell [-config omniql.yaml]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, `  omniql -driver sqlite -dsn :memory: -query '{"target":"t","action":"FIND","filter":{}}'`)
	fmt.Fprintln(os.Stderr, `  omniql -config omniql.yaml -query '{"target":"users","action":"FIND","filter":{}}'`)
	fmt.Fprintln(os.Stderr, `  omniql shell`)
	fmt.Fprintln(os.Stderr, `  omniql shell -config omniql.yaml`)
}

func main() {
	// Detect "shell" subcommand before flag.Parse so it doesn't conflict.
	if len(os.Args) >= 2 && os.Args[1] == "shell" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		runShellMode()
		return
	}

	// Detect "migrate" subcommand
	if len(os.Args) >= 2 && os.Args[1] == "migrate" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		runMigrateMode()
		return
	}

	driverFlag := flag.String("driver", "", "Database driver: sqlite, postgres, mongo, mysql, sqlserver, redis, elasticsearch")
	dsnFlag    := flag.String("dsn",    "", "Connection string / file path / MongoDB URI")
	dbFlag     := flag.String("db",     "", "Database name (required for mongo)")
	queryFlag  := flag.String("query",  "", "JSON-encoded OQL query to execute")
	configFlag := flag.String("config", "omniql.yaml", "Path to omniql.yaml config file")
	flag.Usage = usage
	flag.Parse()

	if *queryFlag == "" {
		usage()
		os.Exit(1)
	}

	var query core.OQLQuery
	if err := json.Unmarshal([]byte(*queryFlag), &query); err != nil {
		log.Fatalf("failed to parse query: %v", err)
	}

	engine := core.NewEngine()

	if *driverFlag != "" {
		// Flag mode: single driver specified on the command line.
		if *dsnFlag == "" {
			log.Fatalf("-dsn is required when -driver is set")
		}
		if err := registerDriver(engine, *driverFlag, *dsnFlag, *dbFlag); err != nil {
			log.Fatal(err)
		}
		engine.Route(query.Target, *driverFlag)
	} else {
		// Config mode: load omniql.yaml and register all drivers.
		cfg, err := LoadConfig(*configFlag)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		if cfg == nil {
			fmt.Fprintf(os.Stderr, "No config file found at %q and no -driver flag provided.\n", *configFlag)
			usage()
			os.Exit(1)
		}
		if err := applyConfig(engine, cfg); err != nil {
			log.Fatal(err)
		}
	}

	result, err := engine.Execute(context.Background(), query)
	if err != nil {
		log.Fatalf("execution error: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}
}

// registerDriver creates and registers a single driver by type name.
func registerDriver(engine *core.Engine, driverType, dsn, db string) error {
	switch driverType {
	case "sqlite":
		drv, err := drvsqlite.New(dsn)
		if err != nil {
			return fmt.Errorf("sqlite: %w", err)
		}
		engine.RegisterDriver(drv)

	case "postgres":
		sqlDB, err := sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("postgres open: %w", err)
		}
		engine.RegisterDriver(drvpostgres.New(sqlDB))

	case "mongo":
		if db == "" {
			return fmt.Errorf("mongo driver requires -db <database name>")
		}
		client, err := mongo.Connect(context.Background(), mongoopts.Client().ApplyURI(dsn))
		if err != nil {
			return fmt.Errorf("mongo connect: %w", err)
		}
		engine.RegisterDriver(drvmongo.New(client, db))

	case "mysql":
		drv, err := drvmysql.New(dsn)
		if err != nil {
			return fmt.Errorf("mysql: %w", err)
		}
		engine.RegisterDriver(drv)

	case "sqlserver":
		drv, err := drvsqlserver.New(dsn)
		if err != nil {
			return fmt.Errorf("sqlserver: %w", err)
		}
		engine.RegisterDriver(drv)

	case "redis":
		drv, err := drvredis.New(dsn)
		if err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		engine.RegisterDriver(drv)

	case "elasticsearch":
		drv, err := drvels.New(dsn)
		if err != nil {
			return fmt.Errorf("elasticsearch: %w", err)
		}
		engine.RegisterDriver(drv)

	default:
		return fmt.Errorf("unknown driver %q: choose sqlite, postgres, mongo, mysql, sqlserver, redis, or elasticsearch", driverType)
	}
	return nil
}

// runShellMode loads config (if present) and starts the interactive REPL.
func runShellMode() {
	configFlag := flag.String("config", "omniql.yaml", "Path to omniql.yaml config file")
	flag.Usage = usage
	flag.Parse()

	engine := core.NewEngine()

	cfg, err := LoadConfig(*configFlag)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg != nil {
		if err := applyConfig(engine, cfg); err != nil {
			log.Fatal(err)
		}
	}

	runREPL(engine)
}

// applyConfig registers all drivers and routes declared in cfg.
func applyConfig(engine *core.Engine, cfg *Config) error {
	// Apply engine options
	core.WithAutoSchema(cfg.Engine.AutoSchema)(engine)

	for name, dc := range cfg.Drivers {
		if err := registerDriver(engine, dc.Type, dc.DSN, dc.DB); err != nil {
			return fmt.Errorf("driver %q: %w", name, err)
		}
	}
	for target, driverKey := range cfg.Routes {
		dc, ok := cfg.Drivers[driverKey]
		if !ok {
			return fmt.Errorf("route %q references unknown driver %q", target, driverKey)
		}
		engine.Route(target, dc.Type)
	}
	return nil
}
