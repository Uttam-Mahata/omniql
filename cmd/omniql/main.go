// Package main is the OmniQL CLI entry-point.
//
// It provides a minimal REPL-style interface for ad-hoc OQL query execution
// against any registered driver.  In production deployments the Core Engine
// is typically embedded via the FFI layer rather than used directly.
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
	drvmongo "github.com/Uttam-Mahata/omniql/pkg/drivers/mongo"
	drvpostgres "github.com/Uttam-Mahata/omniql/pkg/drivers/postgres"
	drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
	_ "github.com/lib/pq" // Postgres database/sql driver
	"go.mongodb.org/mongo-driver/mongo"
	mongoopts "go.mongodb.org/mongo-driver/mongo/options"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: omniql -driver <sqlite|postgres|mongo> -dsn <dsn> [-db <dbname>] -query '<OQL JSON>'")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  -driver   Database driver: sqlite, postgres, or mongo")
	fmt.Fprintln(os.Stderr, "  -dsn      Connection string / file path / MongoDB URI")
	fmt.Fprintln(os.Stderr, "  -db       Database name (required for mongo)")
	fmt.Fprintln(os.Stderr, "  -query    JSON-encoded OQL query to execute")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, `  omniql -driver sqlite -dsn :memory: -query '{"target":"t","action":"FIND","filter":{}}'`)
	fmt.Fprintln(os.Stderr, `  omniql -driver postgres -dsn "postgres://user:pass@localhost/mydb?sslmode=disable" -query '{"target":"users","action":"FIND","filter":{},"options":{"limit":10}}'`)
	fmt.Fprintln(os.Stderr, `  omniql -driver mongo -dsn "mongodb://localhost:27017" -db mydb -query '{"target":"users","action":"FIND","filter":{}}'`)
}

func main() {
	driverFlag := flag.String("driver", "", "Database driver: sqlite, postgres, mongo")
	dsnFlag    := flag.String("dsn",    "", "Connection string / file path / MongoDB URI")
	dbFlag     := flag.String("db",     "", "Database name (required for mongo)")
	queryFlag  := flag.String("query",  "", "JSON-encoded OQL query to execute")
	flag.Usage = usage
	flag.Parse()

	if *queryFlag == "" || *driverFlag == "" || *dsnFlag == "" {
		usage()
		os.Exit(1)
	}

	var query core.OQLQuery
	if err := json.Unmarshal([]byte(*queryFlag), &query); err != nil {
		log.Fatalf("failed to parse query: %v", err)
	}

	engine := core.NewEngine()

	switch *driverFlag {
	case "sqlite":
		drv, err := drvsqlite.New(*dsnFlag)
		if err != nil {
			log.Fatalf("sqlite: %v", err)
		}
		defer drv.Close()
		engine.RegisterDriver(drv)
		engine.Route(query.Target, drv.Name())

	case "postgres":
		db, err := sql.Open("postgres", *dsnFlag)
		if err != nil {
			log.Fatalf("postgres open: %v", err)
		}
		defer db.Close()
		drv := drvpostgres.New(db)
		engine.RegisterDriver(drv)
		engine.Route(query.Target, drv.Name())

	case "mongo":
		if *dbFlag == "" {
			log.Fatalf("mongo driver requires -db <database name>")
		}
		client, err := mongo.Connect(context.Background(), mongoopts.Client().ApplyURI(*dsnFlag))
		if err != nil {
			log.Fatalf("mongo connect: %v", err)
		}
		defer client.Disconnect(context.Background())
		drv := drvmongo.New(client, *dbFlag)
		engine.RegisterDriver(drv)
		engine.Route(query.Target, drv.Name())

	default:
		log.Fatalf("unknown driver %q: choose sqlite, postgres, or mongo", *driverFlag)
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
