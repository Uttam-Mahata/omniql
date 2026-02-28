package core

import (
	"context"
	"fmt"
)

// MigrateOptions configures a cross-driver migration.
type MigrateOptions struct {
	BatchSize int  // records per BATCH_INSERT (default 100)
	DryRun    bool // count only, don't write
}

// MigrateResult reports migration outcome.
type MigrateResult struct {
	RecordsRead    int64 `json:"records_read"`
	RecordsWritten int64 `json:"records_written"`
	Errors         int   `json:"errors"`
}

// Migrate copies all data from sourceTarget (on its routed driver) to
// destTarget (on its routed driver). The two targets can be on different
// drivers — e.g., migrate "users" from Postgres to MongoDB.
func (e *Engine) Migrate(ctx context.Context, sourceTarget, destTarget string, opts MigrateOptions) (*MigrateResult, error) {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}

	srcDriver, err := e.selectDriver(sourceTarget)
	if err != nil {
		return nil, err
	}
	// Verify dest driver exists (even though we use e.Execute later)
	if _, err := e.selectDriver(destTarget); err != nil {
		return nil, err
	}

	result := &MigrateResult{}
	skip := 0

	for {
		// Read batch from source
		rows, _, err := srcDriver.Execute(ctx, OQLQuery{
			Target:  sourceTarget,
			Action:  ActionFind,
			Options: QueryOptions{Limit: opts.BatchSize, Skip: skip},
		})
		if err != nil {
			return result, fmt.Errorf("read error at skip %d: %w", skip, err)
		}
		if len(rows) == 0 {
			break // done
		}

		result.RecordsRead += int64(len(rows))

		if !opts.DryRun {
			_, err := e.Execute(ctx, OQLQuery{
				Target:    destTarget,
				Action:    ActionBatchInsert,
				Documents: rows,
			})

			if err != nil {
				result.Errors++
			} else {
				result.RecordsWritten += int64(len(rows))
			}
		}

		skip += opts.BatchSize
		if len(rows) < opts.BatchSize {
			break // last batch
		}
	}

	return result, nil
}

// MigrateAll copies all data from all targets in sourceDriver to destDriver.
// This is useful for full database migration.
func (e *Engine) MigrateAll(ctx context.Context, sourceDriverName, destDriverName string, opts MigrateOptions) (*MigrateResult, error) {
	srcDriver, err := e.GetDriver(sourceDriverName)
	if err != nil {
		return nil, fmt.Errorf("source driver: %w", err)
	}
	destDriver, err := e.GetDriver(destDriverName)
	if err != nil {
		return nil, fmt.Errorf("dest driver: %w", err)
	}

	targets, err := srcDriver.ListTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}

	totalResult := &MigrateResult{}

	for _, target := range targets {
		// Perform migration for each target
		// We can't use e.Migrate because it relies on routing.
		// We implement a simplified migration loop here using direct driver access.

		skip := 0
		for {
			// Read batch
			rows, _, err := srcDriver.Execute(ctx, OQLQuery{
				Target:  target,
				Action:  ActionFind,
				Options: QueryOptions{Limit: opts.BatchSize, Skip: skip},
			})
			if err != nil {
				return totalResult, fmt.Errorf("read error on target %q skip %d: %w", target, skip, err)
			}
			if len(rows) == 0 {
				break
			}

			totalResult.RecordsRead += int64(len(rows))

			if !opts.DryRun {
				// Ensure schema on first batch if needed
				if skip == 0 {
					if sad, ok := destDriver.(SchemaAwareDriver); ok {
						// Infer schema from the first batch
						// Merge fields from all documents in the batch to get a complete schema
						merged := make(map[string]interface{})
						for _, doc := range rows {
							for k, v := range doc {
								merged[k] = v
							}
						}
						schema := InferSchema(target, merged)
						if err := sad.EnsureTarget(ctx, target, &schema); err != nil {
							return totalResult, fmt.Errorf("ensure target %q: %w", target, err)
						}
					}
				}

				// Write batch
				_, err := destDriver.BatchInsert(ctx, target, rows)
				if err != nil {
					totalResult.Errors++
					// log error? continue?
				} else {
					totalResult.RecordsWritten += int64(len(rows))
				}
			}

			skip += opts.BatchSize
			if len(rows) < opts.BatchSize {
				break
			}
		}
	}

	return totalResult, nil
}
