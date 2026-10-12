package protection

import "github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"

// Engine runs every configured element against each reading.
type Engine struct {
	elements []Element
}

// NewEngine accepts any number of elements (a "variadic" parameter).
func NewEngine(elements ...Element) *Engine {
	return &Engine{elements: elements}
}

// Evaluate returns every event raised by every element for one reading.
func (e *Engine) Evaluate(r grid.Reading) []Event {
	var events []Event
	for _, el := range e.elements {
		events = append(events, el.Check(r)...)
	}
	return events
}

// DefaultElements returns typical settings based on nominal voltage and current.
func DefaultElements(nominalV, nominalI float64) []Element {
	return []Element{
		Undervoltage{PickupV: 0.88 * nominalV},
		Overvoltage{PickupV: 1.10 * nominalV},
		Overcurrent{PickupA: 2.0 * nominalI},
		Frequency{MinHz: 59.5, MaxHz: 60.5},
		VoltageImbalance{MaxPercent: 2.0},
	}
}

// TransitionKind says whether a condition started or ended.
type TransitionKind string

const (
	Pickup  TransitionKind = "PICKUP"
	Dropout TransitionKind = "DROPOUT"
)

// Transition is a change in state: a condition starting or clearing.
type Transition struct {
	Kind  TransitionKind
	Event Event
}

// Monitor remembers which conditions are active so it reports each one
// once when it picks up and once when it drops out, instead of every second.
type Monitor struct {
	engine *Engine
	active map[string]Event
}

func NewMonitor(engine *Engine) *Monitor {
	return &Monitor{engine: engine, active: make(map[string]Event)}
}

// Process evaluates one reading and returns only the state changes.
func (m *Monitor) Process(r grid.Reading) []Transition {
	current := make(map[string]Event)
	for _, ev := range m.engine.Evaluate(r) {
		current[ev.Key()] = ev
	}

	var out []Transition

	// New conditions → PICKUP
	for key, ev := range current {
		if _, wasActive := m.active[key]; !wasActive {
			out = append(out, Transition{Kind: Pickup, Event: ev})
		}
		m.active[key] = ev
	}

	// Conditions on this device that are no longer present → DROPOUT
	for key, ev := range m.active {
		if ev.DeviceID != r.DeviceID {
			continue // other relays are handled by their own readings
		}
		if _, stillActive := current[key]; !stillActive {
			ev.Timestamp = r.Timestamp
			out = append(out, Transition{Kind: Dropout, Event: ev})
			delete(m.active, key)
		}
	}

	return out
}
