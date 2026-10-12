package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/simulator"
)

func main() {
	// Stop on Ctrl+C, or automatically after 30 seconds.
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

	// When every relay has stopped, close the channel so the loop below ends.
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
		}
	}
	fmt.Println("Simulator stopped.")
}
