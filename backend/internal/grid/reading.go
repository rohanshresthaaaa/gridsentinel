package grid

import (
	"math"
	"time"
)

// PhaseReading holds the measurements for a single phase.
type PhaseReading struct {
	VoltageV float64 // RMS voltage, phase-to-neutral (volts)
	CurrentA float64 // RMS current (amps)
	AngleDeg float64 // voltage phase angle (degrees)
}

// Reading is one snapshot from a relay, covering all three phases.
type Reading struct {
	DeviceID    string
	Timestamp   time.Time
	PhaseA      PhaseReading
	PhaseB      PhaseReading
	PhaseC      PhaseReading
	FrequencyHz float64
}

// AverageVoltage returns the mean of the three phase voltages.
func (r Reading) AverageVoltage() float64 {
	return (r.PhaseA.VoltageV + r.PhaseB.VoltageV + r.PhaseC.VoltageV) / 3
}

// VoltageImbalancePercent uses the NEMA definition:
// (max deviation from average / average) * 100.
func (r Reading) VoltageImbalancePercent() float64 {
	avg := r.AverageVoltage()
	if avg == 0 {
		return 0
	}
	maxDev := 0.0
	for _, v := range []float64{r.PhaseA.VoltageV, r.PhaseB.VoltageV, r.PhaseC.VoltageV} {
		dev := math.Abs(v - avg)
		if dev > maxDev {
			maxDev = dev
		}
	}
	return maxDev / avg * 100
}
