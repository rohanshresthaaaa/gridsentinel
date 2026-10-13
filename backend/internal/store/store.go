package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store saves and loads protection events from PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// Alarm is one stored protection event. DroppedOutAt is nil while active.
type Alarm struct {
	ID           int64
	DeviceID     string
	Element      string
	Name         string
	Phase        string
	Value        float64
	Threshold    float64
	Unit         string
	PickedUpAt   time.Time
	DroppedOutAt *time.Time
}

// New connects to the database and checks that it is reachable.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases all database connections.
func (s *Store) Close() {
	s.pool.Close()
}

// Migrate applies any migration files that have not been applied yet.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations") // sorted by name
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, entry := range entries {
		var applied bool
		err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`,
			entry.Name()).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		if err := s.applyMigration(ctx, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// applyMigration runs one migration inside a transaction: all of it, or none of it.
func (s *Store) applyMigration(ctx context.Context, name string) error {
	sqlBytes, err := migrationFS.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // does nothing if Commit already succeeded

	// A migration file holds several SQL statements. The low-level PgConn.Exec
	// sends them all at once using PostgreSQL's simple protocol.
	if _, err := tx.Conn().PgConn().Exec(ctx, string(sqlBytes)).ReadAll(); err != nil {
		return fmt.Errorf("run migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return tx.Commit(ctx)
}

// RecordPickup stores a new active alarm. If the same condition is already
// active, the unique index rejects the duplicate and nothing happens.
func (s *Store) RecordPickup(ctx context.Context, ev protection.Event) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO protection_events
			(device_id, element, name, phase, value, threshold, unit, picked_up_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`,
		ev.DeviceID, ev.Element, ev.Name, ev.Phase, ev.Value, ev.Threshold, ev.Unit, ev.Timestamp)
	if err != nil {
		return fmt.Errorf("insert pickup: %w", err)
	}
	return nil
}

// RecordDropout closes the matching active alarm.
func (s *Store) RecordDropout(ctx context.Context, ev protection.Event) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE protection_events
		SET dropped_out_at = $4
		WHERE device_id = $1 AND element = $2 AND phase = $3
		  AND dropped_out_at IS NULL`,
		ev.DeviceID, ev.Element, ev.Phase, ev.Timestamp)
	if err != nil {
		return fmt.Errorf("update dropout: %w", err)
	}
	return nil
}

// CloseStaleAlarms ends alarms left active by a previous run. After a restart
// we can no longer vouch that they are still active.
func (s *Store) CloseStaleAlarms(ctx context.Context, at time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE protection_events SET dropped_out_at = $1 WHERE dropped_out_at IS NULL`, at)
	if err != nil {
		return 0, fmt.Errorf("close stale alarms: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ActiveAlarms returns all alarms that have not dropped out, oldest first.
func (s *Store) ActiveAlarms(ctx context.Context) ([]Alarm, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, device_id, element, name, phase, value, threshold, unit,
		       picked_up_at, dropped_out_at
		FROM protection_events
		WHERE dropped_out_at IS NULL
		ORDER BY picked_up_at`)
	if err != nil {
		return nil, fmt.Errorf("query active alarms: %w", err)
	}
	defer rows.Close()

	var alarms []Alarm
	for rows.Next() {
		var a Alarm
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Element, &a.Name, &a.Phase,
			&a.Value, &a.Threshold, &a.Unit, &a.PickedUpAt, &a.DroppedOutAt); err != nil {
			return nil, fmt.Errorf("scan alarm: %w", err)
		}
		alarms = append(alarms, a)
	}
	return alarms, rows.Err()
}
