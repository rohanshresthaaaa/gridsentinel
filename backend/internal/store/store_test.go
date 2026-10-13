package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
)

// testStore connects to the TEST database and empties it.
// Tests are skipped if the database URL is not set, so `go test ./...`
// still works on machines without Docker.
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("GRIDSENTINEL_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set GRIDSENTINEL_TEST_DATABASE_URL to run database tests")
	}

	ctx := context.Background()
	s, err := New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(s.Close)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := s.pool.Exec(ctx, "TRUNCATE protection_events RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

func undervoltageEvent(device string, at time.Time) protection.Event {
	return protection.Event{
		DeviceID: device, Timestamp: at,
		Element: "27", Name: "Undervoltage", Phase: "C",
		Value: 96, Threshold: 105.6, Unit: "V",
	}
}

func TestMigrateIsSafeToRunTwice(t *testing.T) {
	s := testStore(t) // already migrated once
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}

func TestPickupThenDropout(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC().Truncate(time.Microsecond) // PostgreSQL stores microseconds

	ev := undervoltageEvent("RELAY-01", t0)
	if err := s.RecordPickup(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPickup(ctx, ev); err != nil { // duplicate must be ignored
		t.Fatalf("duplicate pickup should not error: %v", err)
	}

	active, err := s.ActiveAlarms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("want 1 active alarm, got %d", len(active))
	}
	a := active[0]
	if a.Element != "27" || a.Phase != "C" || !a.PickedUpAt.Equal(t0) || a.DroppedOutAt != nil {
		t.Fatalf("stored alarm is wrong: %+v", a)
	}

	ev.Timestamp = t0.Add(5 * time.Second)
	if err := s.RecordDropout(ctx, ev); err != nil {
		t.Fatal(err)
	}
	active, err = s.ActiveAlarms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("want 0 active alarms after dropout, got %d", len(active))
	}
}

func TestCloseStaleAlarms(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now()

	if err := s.RecordPickup(ctx, undervoltageEvent("RELAY-01", now)); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPickup(ctx, undervoltageEvent("RELAY-02", now)); err != nil {
		t.Fatal(err)
	}

	n, err := s.CloseStaleAlarms(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 alarms closed, got %d", n)
	}
}
