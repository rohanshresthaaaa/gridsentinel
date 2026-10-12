package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/simulator"
)

func main() {
	// Stop on Ctrl+C, or automatically after 15 seconds.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
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

	fmt.Println("GridSentinel simulator running (Ctrl+C to stop)...")
	for r := range readings {
		fmt.Printf("%s %s  Va=%6.1f Vb=%6.1f Vc=%6.1f  Ia=%6.1f A  f=%.3f Hz  imbalance=%5.2f%%\n",
			r.Timestamp.Format("15:04:05"), r.DeviceID,
			r.PhaseA.VoltageV, r.PhaseB.VoltageV, r.PhaseC.VoltageV,
			r.PhaseA.CurrentA, r.FrequencyHz, r.VoltageImbalancePercent())
	}
	fmt.Println("Simulator stopped.")
}
