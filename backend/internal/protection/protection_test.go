package protection

import (
	"slices"
	"testing"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
)

// reading builds a test reading. Phases B and C carry nominal 50 A.
func reading(va, vb, vc, ia, freq float64) grid.Reading {
	return grid.Reading{
		DeviceID:    "TEST-01",
		Timestamp:   time.Now(),
		PhaseA:      grid.PhaseReading{VoltageV: va, CurrentA: ia},
		PhaseB:      grid.PhaseReading{VoltageV: vb, CurrentA: 50},
		PhaseC:      grid.PhaseReading{VoltageV: vc, CurrentA: 50},
		FrequencyHz: freq,
	}
}

func codes(events []Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Element)
	}
	return out
}

func TestEngineDetectsFaults(t *testing.T) {
	engine := NewEngine(DefaultElements(120, 50)...)

	tests := []struct {
		name string
		r    grid.Reading
		want []string
	}{
		{"healthy system", reading(120, 120, 120, 50, 60), nil},
		{"phase C sag", reading(120, 120, 96, 50, 60), []string{"27", "47"}},
		{"balanced three-phase sag", reading(100, 100, 100, 50, 60), []string{"27", "27", "27"}},
		{"phase A swell", reading(135, 120, 120, 50, 60), []string{"59", "47"}},
		{"phase A overcurrent", reading(120, 120, 120, 300, 60), []string{"50"}},
		{"underfrequency", reading(120, 120, 120, 50, 59.2), []string{"81U"}},
		{"overfrequency", reading(120, 120, 120, 50, 60.8), []string{"81O"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codes(engine.Evaluate(tc.r))
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestElementReportsCorrectPhase(t *testing.T) {
	events := Undervoltage{PickupV: 105}.Check(reading(120, 90, 120, 50, 60))
	if len(events) != 1 || events[0].Phase != "B" {
		t.Fatalf("expected one event on phase B, got %+v", events)
	}
}

func TestValueAtPickupDoesNotOperate(t *testing.T) {
	events := Undervoltage{PickupV: 105}.Check(reading(105, 105, 105, 50, 60))
	if len(events) != 0 {
		t.Fatalf("voltage exactly at pickup should not operate, got %+v", events)
	}
}

func countKinds(ts []Transition) (pickups, dropouts int) {
	for _, tr := range ts {
		if tr.Kind == Pickup {
			pickups++
		} else {
			dropouts++
		}
	}
	return
}

func TestMonitorReportsEachConditionOnce(t *testing.T) {
	m := NewMonitor(NewEngine(DefaultElements(120, 50)...))
	healthy := reading(120, 120, 120, 50, 60)
	sag := reading(120, 120, 96, 50, 60) // raises 27 and 47

	steps := []struct {
		name                      string
		r                         grid.Reading
		wantPickups, wantDropouts int
	}{
		{"healthy start", healthy, 0, 0},
		{"sag begins", sag, 2, 0},
		{"sag continues", sag, 0, 0},
		{"sag clears", healthy, 0, 2},
		{"healthy again", healthy, 0, 0},
	}

	for _, s := range steps {
		p, d := countKinds(m.Process(s.r))
		if p != s.wantPickups || d != s.wantDropouts {
			t.Errorf("%s: got %d pickups / %d dropouts, want %d / %d",
				s.name, p, d, s.wantPickups, s.wantDropouts)
		}
	}
}

func TestMonitorKeepsDevicesSeparate(t *testing.T) {
	m := NewMonitor(NewEngine(DefaultElements(120, 50)...))

	sag := reading(120, 120, 96, 50, 60) // TEST-01 faulted
	other := reading(120, 120, 120, 50, 60)
	other.DeviceID = "TEST-02" // a different, healthy relay

	m.Process(sag)
	if got := m.Process(other); len(got) != 0 {
		t.Fatalf("healthy TEST-02 must not clear TEST-01's alarms, got %+v", got)
	}

	healthy := reading(120, 120, 120, 50, 60)
	if _, d := countKinds(m.Process(healthy)); d != 2 {
		t.Fatalf("expected TEST-01's 2 alarms to drop out, got %d", d)
	}
}
