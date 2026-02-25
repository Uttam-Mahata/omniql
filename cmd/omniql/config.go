package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DriverConfig holds the connection configuration for a named driver.
type DriverConfig struct {
	Type string `yaml:"type"` // sqlite, postgres, mongo, mysql, sqlserver, redis, elasticsearch
	DSN  string `yaml:"dsn"`
	DB   string `yaml:"db,omitempty"` // database name (required for mongo)
}

// Config is the top-level structure for omniql.yaml.
//
// Example:
//
//	drivers:
//	  main_db: { type: postgres, dsn: "postgres://..." }
//	  cache:   { type: sqlite,   dsn: "./cache.db" }
//	routes:
//	  users:    main_db
//	  sessions: cache
type Config struct {
	Drivers map[string]DriverConfig `yaml:"drivers"`
	Routes  map[string]string       `yaml:"routes"` // target -> driver name key
}

// LoadConfig reads an omniql.yaml file from the given path and returns the
// parsed Config.  Returns (nil, nil) if the file does not exist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	return &cfg, nil
}
