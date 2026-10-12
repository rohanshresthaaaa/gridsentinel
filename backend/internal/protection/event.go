package protection

import (
	"fmt"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
)

// Event describes one protection element operating on one reading.
type Event struct {
	DeviceID  string
	Timestamp time.Time
	Element   string  // ANSI device number, e.g. "27"
	Name      string  // e.g. "Undervoltage"
	Phase     string  // "A", "B", "C", or "" if not phase-specific
	Value     float64 // measured value that caused the operation
	Threshold float64 // the setting that was crossed
	Unit      string  // "V", "A", "Hz", or "%"
}

// Key uniquely identifies a condition: same device, same element, same phase.
func (e Event) Key() string {
	return e.DeviceID + "|" + e.Element + "|" + e.Phase
}

func (e Event) String() string {
	phase := ""
	if e.Phase != "" {
		phase = " phase " + e.Phase
	}
	return fmt.Sprintf("%s %s%s: %.2f %s (setting %.2f %s)",
		e.Element, e.Name, phase, e.Value, e.Unit, e.Threshold, e.Unit)
}

// Element is one protection function. Anything with a Check method
// that matches this signature is an Element.
type Element interface {
	Check(r grid.Reading) []Event
}
