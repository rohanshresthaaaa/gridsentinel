package grid

import (
	"math"
	"testing"
)

func TestVoltageImbalancePercent(t *testing.T) {
	tests := []struct {
		name     string
		a, b, c  float64
		expected float64
	}{
		{"balanced system", 120, 120, 120, 0},
		{"phase C sagging", 120, 120, 114, 3.3898},
		{"all zero (de-energized)", 0, 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Reading{
				PhaseA: PhaseReading{VoltageV: tc.a},
				PhaseB: PhaseReading{VoltageV: tc.b},
				PhaseC: PhaseReading{VoltageV: tc.c},
			}
			got := r.VoltageImbalancePercent()
			if math.Abs(got-tc.expected) > 0.001 {
				t.Errorf("got %.4f, want %.4f", got, tc.expected)
			}
		})
	}
}
