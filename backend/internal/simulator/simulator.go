package simulator

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
)

// FaultKind identifies a type of power system disturbance.
type FaultKind int

const (
	FaultNone           FaultKind = iota // 0
	FaultVoltageSag                      // 1: one phase drops (e.g. a fault on a nearby line)
	FaultOvercurrent                     // 2: phase A current spikes (e.g. a short circuit)
	FaultFrequencyDrift                  // 3: generation can't keep up with load
)

// String makes FaultKind print as a readable name.
func (f FaultKind) String() string {
	switch f {
	case FaultVoltageSag:
		return "VoltageSag"
	case FaultOvercurrent:
		return "Overcurrent"
	case FaultFrequencyDrift:
		return "FrequencyDrift"
	default:
		return "None"
	}
}

// Config describes one simulated relay.
type Config struct {
	DeviceID       string
	NominalVoltage float64 // phase-to-neutral volts, e.g. 120
	NominalCurrent float64 // amps, e.g. 50
	FaultChance    float64 // chance (0 to 1) of starting a fault on each reading
	Seed           uint64  // random seed; same seed = same sequence (useful for tests)
}

// Relay simulates one protective relay measuring a 3-phase circuit.
type Relay struct {
	cfg        Config
	rng        *rand.Rand
	fault      FaultKind
	faultTicks int // how many more readings the current fault lasts
}

// NewRelay creates a relay from a config.
func NewRelay(cfg Config) *Relay {
	return &Relay{
		cfg: cfg,
		rng: rand.New(rand.NewPCG(cfg.Seed, cfg.Seed+1)),
	}
}

// Fault returns the fault currently active on this relay.
func (r *Relay) Fault() FaultKind { return r.fault }

// ForceFault starts a fault immediately, lasting the given number of readings.
func (r *Relay) ForceFault(kind FaultKind, ticks int) {
	r.fault = kind
	r.faultTicks = ticks
}

// noise returns a multiplier close to 1.0, e.g. 0.998 or 1.003.
func (r *Relay) noise(pct float64) float64 {
	return 1 + r.rng.NormFloat64()*pct
}

// updateFault counts down an active fault, or randomly starts a new one.
func (r *Relay) updateFault() {
	if r.faultTicks > 0 {
		r.faultTicks--
		return
	}
	r.fault = FaultNone
	if r.rng.Float64() < r.cfg.FaultChance {
		r.fault = FaultKind(1 + r.rng.IntN(3)) // random fault 1..3
		r.faultTicks = 3 + r.rng.IntN(5)       // lasts a few seconds
	}
}

// Next produces one reading. It is separate from Run so it can be tested
// without waiting on real time.
func (r *Relay) Next(now time.Time) grid.Reading {
	r.updateFault()

	v := r.cfg.NominalVoltage
	i := r.cfg.NominalCurrent

	reading := grid.Reading{
		DeviceID:    r.cfg.DeviceID,
		Timestamp:   now,
		PhaseA:      grid.PhaseReading{VoltageV: v * r.noise(0.002), CurrentA: i * r.noise(0.01), AngleDeg: 0},
		PhaseB:      grid.PhaseReading{VoltageV: v * r.noise(0.002), CurrentA: i * r.noise(0.01), AngleDeg: -120},
		PhaseC:      grid.PhaseReading{VoltageV: v * r.noise(0.002), CurrentA: i * r.noise(0.01), AngleDeg: 120},
		FrequencyHz: 60 + r.rng.NormFloat64()*0.005,
	}

	switch r.fault {
	case FaultVoltageSag:
		reading.PhaseC.VoltageV *= 0.80
	case FaultOvercurrent:
		reading.PhaseA.CurrentA *= 6
	case FaultFrequencyDrift:
		reading.FrequencyHz -= 0.8
	}

	return reading
}

// Run sends a reading to out every interval until ctx is cancelled.
// It is meant to be started as a goroutine: go relay.Run(ctx, time.Second, ch)
func (r *Relay) Run(ctx context.Context, interval time.Duration, out chan<- grid.Reading) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			select {
			case out <- r.Next(now):
			case <-ctx.Done():
				return
			}
		}
	}
}
