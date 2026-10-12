package protection

import "github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"

// Compile-time checks: the build fails if any type stops satisfying Element.
var (
	_ Element = Undervoltage{}
	_ Element = Overvoltage{}
	_ Element = Overcurrent{}
	_ Element = Frequency{}
	_ Element = VoltageImbalance{}
)

type namedPhase struct {
	name string
	p    grid.PhaseReading
}

func phasesOf(r grid.Reading) []namedPhase {
	return []namedPhase{{"A", r.PhaseA}, {"B", r.PhaseB}, {"C", r.PhaseC}}
}

func newEvent(r grid.Reading, element, name, phase string, value, threshold float64, unit string) Event {
	return Event{
		DeviceID: r.DeviceID, Timestamp: r.Timestamp,
		Element: element, Name: name, Phase: phase,
		Value: value, Threshold: threshold, Unit: unit,
	}
}

// Undervoltage (ANSI 27) operates when any phase voltage falls below PickupV.
type Undervoltage struct{ PickupV float64 }

func (u Undervoltage) Check(r grid.Reading) []Event {
	var events []Event
	for _, ph := range phasesOf(r) {
		if ph.p.VoltageV < u.PickupV {
			events = append(events, newEvent(r, "27", "Undervoltage", ph.name, ph.p.VoltageV, u.PickupV, "V"))
		}
	}
	return events
}

// Overvoltage (ANSI 59) operates when any phase voltage rises above PickupV.
type Overvoltage struct{ PickupV float64 }

func (o Overvoltage) Check(r grid.Reading) []Event {
	var events []Event
	for _, ph := range phasesOf(r) {
		if ph.p.VoltageV > o.PickupV {
			events = append(events, newEvent(r, "59", "Overvoltage", ph.name, ph.p.VoltageV, o.PickupV, "V"))
		}
	}
	return events
}

// Overcurrent (ANSI 50) operates instantly when any phase current exceeds PickupA.
type Overcurrent struct{ PickupA float64 }

func (o Overcurrent) Check(r grid.Reading) []Event {
	var events []Event
	for _, ph := range phasesOf(r) {
		if ph.p.CurrentA > o.PickupA {
			events = append(events, newEvent(r, "50", "Overcurrent", ph.name, ph.p.CurrentA, o.PickupA, "A"))
		}
	}
	return events
}

// Frequency (ANSI 81U/81O) operates when frequency leaves the MinHz..MaxHz band.
type Frequency struct{ MinHz, MaxHz float64 }

func (f Frequency) Check(r grid.Reading) []Event {
	switch {
	case r.FrequencyHz < f.MinHz:
		return []Event{newEvent(r, "81U", "Underfrequency", "", r.FrequencyHz, f.MinHz, "Hz")}
	case r.FrequencyHz > f.MaxHz:
		return []Event{newEvent(r, "81O", "Overfrequency", "", r.FrequencyHz, f.MaxHz, "Hz")}
	}
	return nil
}

// VoltageImbalance (approximating ANSI 47) operates when NEMA imbalance exceeds MaxPercent.
type VoltageImbalance struct{ MaxPercent float64 }

func (v VoltageImbalance) Check(r grid.Reading) []Event {
	imb := r.VoltageImbalancePercent()
	if imb > v.MaxPercent {
		return []Event{newEvent(r, "47", "Voltage Imbalance", "", imb, v.MaxPercent, "%")}
	}
	return nil
}
