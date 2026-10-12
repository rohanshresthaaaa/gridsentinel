package simulator

import (
	"context"
	"testing"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
)

func newTestRelay() *Relay {
	return NewRelay(Config{
		DeviceID:       "TEST-01",
		NominalVoltage: 120,
		NominalCurrent: 50,
		FaultChance:    0, // no random faults in tests
		Seed:           42,
	})
}

func TestNormalReadingsStayNearNominal(t *testing.T) {
	r := newTestRelay()
	for i := 0; i < 100; i++ {
		rd := r.Next(time.Now())
		for _, v := range []float64{rd.PhaseA.VoltageV, rd.PhaseB.VoltageV, rd.PhaseC.VoltageV} {
			if v < 117.6 || v > 122.4 { // within 2% of 120 V
				t.Fatalf("reading %d: voltage %.2f out of normal range", i, v)
			}
		}
		if rd.FrequencyHz < 59.95 || rd.FrequencyHz > 60.05 {
			t.Fatalf("reading %d: frequency %.3f out of normal range", i, rd.FrequencyHz)
		}
		if rd.VoltageImbalancePercent() > 1 {
			t.Fatalf("reading %d: imbalance %.2f%% too high for a healthy system", i, rd.VoltageImbalancePercent())
		}
	}
}

func TestForcedFaultsAffectReadings(t *testing.T) {
	tests := []struct {
		kind  FaultKind
		check func(grid.Reading) bool
	}{
		{FaultVoltageSag, func(rd grid.Reading) bool { return rd.PhaseC.VoltageV < 0.85*120 }},
		{FaultOvercurrent, func(rd grid.Reading) bool { return rd.PhaseA.CurrentA > 3*50 }},
		{FaultFrequencyDrift, func(rd grid.Reading) bool { return rd.FrequencyHz < 59.5 }},
	}

	for _, tc := range tests {
		t.Run(tc.kind.String(), func(t *testing.T) {
			r := newTestRelay()
			r.ForceFault(tc.kind, 3)
			rd := r.Next(time.Now())
			if !tc.check(rd) {
				t.Errorf("%s did not show up in reading: %+v", tc.kind, rd)
			}
		})
	}
}

func TestFaultClearsAfterTicks(t *testing.T) {
	r := newTestRelay()
	r.ForceFault(FaultVoltageSag, 2)

	r.Next(time.Now())
	r.Next(time.Now())
	if r.Fault() != FaultVoltageSag {
		t.Fatalf("fault should still be active after 2 readings, got %s", r.Fault())
	}

	r.Next(time.Now())
	if r.Fault() != FaultNone {
		t.Fatalf("fault should have cleared, got %s", r.Fault())
	}
}

func TestRunStopsWhenCancelled(t *testing.T) {
	r := newTestRelay()
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan grid.Reading)
	done := make(chan struct{})

	go func() {
		r.Run(ctx, 10*time.Millisecond, out)
		close(done)
	}()

	<-out    // wait for at least one reading
	cancel() // tell Run to stop

	select {
	case <-done:
		// success: Run returned
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context was cancelled")
	}
}
