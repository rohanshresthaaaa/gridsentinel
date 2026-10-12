package main

import (
	"fmt"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
)

func main() {
	r := grid.Reading{
		DeviceID:    "RELAY-01",
		Timestamp:   time.Now(),
		PhaseA:      grid.PhaseReading{VoltageV: 120.0, CurrentA: 50, AngleDeg: 0},
		PhaseB:      grid.PhaseReading{VoltageV: 119.5, CurrentA: 51, AngleDeg: -120},
		PhaseC:      grid.PhaseReading{VoltageV: 112.0, CurrentA: 49, AngleDeg: 120},
		FrequencyHz: 60.0,
	}
	fmt.Printf("%s avg voltage: %.2f V, imbalance: %.2f%%\n",
		r.DeviceID, r.AverageVoltage(), r.VoltageImbalancePercent())
}