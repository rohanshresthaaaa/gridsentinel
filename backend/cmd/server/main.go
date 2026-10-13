package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/simulator"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/store"
)

const defaultDatabaseURL = "postgres://gridsentinel:gridsentinel@localhost:5433/gridsentinel?sslmode=disable"

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
	}

	// --- Database startup ---
	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startCancel()

	db, err := store.New(startCtx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v (is it running? try: docker compose up -d)", err)
	}
	defer db.Close()

	if err := db.Migrate(startCtx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if n, err := db.CloseStaleAlarms(startCtx, time.Now()); err != nil {
		log.Fatalf("close stale alarms: %v", err)
	} else if n > 0 {
		fmt.Printf("Closed %d alarm(s) left active by a previous run\n", n)
	}

	// --- Simulation ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	readings := make(chan grid.Reading, 16)
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		relay := simulator.NewRelay(simulator.Config{
			DeviceID:       fmt.Sprintf("RELAY-%02d", i),
			NominalVoltage: 120,
			NominalCurrent: 50,
			FaultChance:    0.1,
			Seed:           uint64(time.Now().UnixNano()) + uint64(i),
		})
		wg.Go(func() {
			relay.Run(ctx, time.Second, readings)
		})
	}

	go func() {
		wg.Wait()
		close(readings)
	}()

	monitor := protection.NewMonitor(protection.NewEngine(protection.DefaultElements(120, 50)...))

	fmt.Println("GridSentinel protection running (Ctrl+C to stop)...")
	for r := range readings {
		for _, tr := range monitor.Process(r) {
			fmt.Printf("%s %s  %-7s %s\n",
				tr.Event.Timestamp.Format("15:04:05"), tr.Event.DeviceID, tr.Kind, tr.Event)
			if err := save(db, tr); err != nil {
				log.Printf("save %s: %v", tr.Kind, err)
			}
		}
	}

	fmt.Println("Simulator stopped.")
	printActiveAlarms(db)
}

// save writes one transition. It uses its own timeout, so writes still work
// while the program is shutting down after the main context is cancelled.
func save(db *store.Store, tr protection.Transition) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if tr.Kind == protection.Pickup {
		return db.RecordPickup(ctx, tr.Event)
	}
	return db.RecordDropout(ctx, tr.Event)
}

func printActiveAlarms(db *store.Store) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	alarms, err := db.ActiveAlarms(ctx)
	if err != nil {
		log.Printf("load active alarms: %v", err)
		return
	}
	fmt.Printf("\nActive alarms in database: %d\n", len(alarms))
	for _, a := range alarms {
		fmt.Printf("  %s  %s %s phase=%q since %s\n",
			a.DeviceID, a.Element, a.Name, a.Phase, a.PickedUpAt.Format("15:04:05"))
	}
}
