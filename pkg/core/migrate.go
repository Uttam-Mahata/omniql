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
			// Write batch to dest.
			// destDriver.BatchInsert might not be implemented by all drivers (e.g. if I missed one),
			// but interface defines it.

			// We might need to ensure target on first batch?
			// The engine handles EnsureTarget for Execute, but BatchInsert is called directly on driver here.
			// So we should call EnsureTarget if needed.

			// If destDriver implements SchemaAwareDriver, we should call EnsureTarget.
			// But wait, Engine.Execute does this.
			// Should we use Engine.Execute(BATCH_INSERT) instead of calling driver directly?
			// Yes, that would reuse the EnsureTarget logic.

			// However, Engine.Execute expects OQLQuery.
			// Using Engine.Execute handles validation and EnsureTarget.

			_, err := e.Execute(ctx, OQLQuery{
				Target:    destTarget,
				Action:    ActionBatchInsert,
				Documents: rows,
			})

			if err != nil {
				result.Errors++
				// continue with next batch? Or stop?
				// Plan says: "Errors++ // continue with next batch"
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
