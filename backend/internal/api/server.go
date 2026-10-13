package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/store"
)

// AlarmStore is the part of the database the API needs.
// *store.Store satisfies it, and tests use a small fake instead.
type AlarmStore interface {
	ActiveAlarms(ctx context.Context) ([]store.Alarm, error)
	RecentAlarms(ctx context.Context, deviceID string, limit int) ([]store.Alarm, error)
}

// Server serves the REST API and the live event stream.
type Server struct {
	store AlarmStore
	hub   *Hub

	mu     sync.RWMutex
	latest map[string]grid.Reading // newest reading per device
}

func NewServer(s AlarmStore, hub *Hub) *Server {
	return &Server{store: s, hub: hub, latest: make(map[string]grid.Reading)}
}

// ---------- JSON shapes sent to the browser (DTOs) ----------

type phaseJSON struct {
	VoltageV float64 `json:"voltageV"`
	CurrentA float64 `json:"currentA"`
	AngleDeg float64 `json:"angleDeg"`
}

type readingJSON struct {
	DeviceID         string    `json:"deviceId"`
	Timestamp        time.Time `json:"timestamp"`
	PhaseA           phaseJSON `json:"phaseA"`
	PhaseB           phaseJSON `json:"phaseB"`
	PhaseC           phaseJSON `json:"phaseC"`
	FrequencyHz      float64   `json:"frequencyHz"`
	ImbalancePercent float64   `json:"imbalancePercent"`
}

type alarmJSON struct {
	ID           int64      `json:"id"`
	DeviceID     string     `json:"deviceId"`
	Element      string     `json:"element"`
	Name         string     `json:"name"`
	Phase        string     `json:"phase"`
	Value        float64    `json:"value"`
	Threshold    float64    `json:"threshold"`
	Unit         string     `json:"unit"`
	PickedUpAt   time.Time  `json:"pickedUpAt"`
	DroppedOutAt *time.Time `json:"droppedOutAt"`
	Active       bool       `json:"active"`
}

type transitionJSON struct {
	Kind      string    `json:"kind"`
	DeviceID  string    `json:"deviceId"`
	Element   string    `json:"element"`
	Name      string    `json:"name"`
	Phase     string    `json:"phase"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

func toPhaseJSON(p grid.PhaseReading) phaseJSON {
	return phaseJSON{VoltageV: p.VoltageV, CurrentA: p.CurrentA, AngleDeg: p.AngleDeg}
}

func toReadingJSON(r grid.Reading) readingJSON {
	return readingJSON{
		DeviceID:         r.DeviceID,
		Timestamp:        r.Timestamp,
		PhaseA:           toPhaseJSON(r.PhaseA),
		PhaseB:           toPhaseJSON(r.PhaseB),
		PhaseC:           toPhaseJSON(r.PhaseC),
		FrequencyHz:      r.FrequencyHz,
		ImbalancePercent: r.VoltageImbalancePercent(),
	}
}

func toAlarmsJSON(alarms []store.Alarm) []alarmJSON {
	out := make([]alarmJSON, 0, len(alarms)) // never nil, so JSON is [] not null
	for _, a := range alarms {
		out = append(out, alarmJSON{
			ID: a.ID, DeviceID: a.DeviceID, Element: a.Element, Name: a.Name,
			Phase: a.Phase, Value: a.Value, Threshold: a.Threshold, Unit: a.Unit,
			PickedUpAt: a.PickedUpAt, DroppedOutAt: a.DroppedOutAt,
			Active: a.DroppedOutAt == nil,
		})
	}
	return out
}

// ---------- Publishing (called by main for every reading and transition) ----------

// PublishReading remembers the newest reading for its device and streams it live.
func (s *Server) PublishReading(r grid.Reading) {
	s.mu.Lock()
	s.latest[r.DeviceID] = r
	s.mu.Unlock()
	s.publish("reading", toReadingJSON(r))
}

// PublishTransition streams an alarm pickup or dropout live.
func (s *Server) PublishTransition(tr protection.Transition) {
	ev := tr.Event
	s.publish("alarm", transitionJSON{
		Kind: string(tr.Kind), DeviceID: ev.DeviceID, Element: ev.Element,
		Name: ev.Name, Phase: ev.Phase, Value: ev.Value, Threshold: ev.Threshold,
		Unit: ev.Unit, Timestamp: ev.Timestamp,
	})
}

// publish formats one Server-Sent Event and hands it to the hub.
func (s *Server) publish(event string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("encode %s event: %v", event, err)
		return
	}
	s.hub.Publish([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)))
}

// ---------- Routes ----------

// Handler returns the HTTP handler with every route registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/devices", s.handleDevices)
	mux.HandleFunc("GET /api/alarms/active", s.handleActiveAlarms)
	mux.HandleFunc("GET /api/alarms", s.handleAlarms)
	mux.HandleFunc("GET /api/stream", s.handleStream)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"streamClients": s.hub.Count(),
	})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	devices := make([]readingJSON, 0, len(s.latest))
	for _, reading := range s.latest {
		devices = append(devices, toReadingJSON(reading))
	}
	s.mu.RUnlock()

	slices.SortFunc(devices, func(a, b readingJSON) int {
		return strings.Compare(a.DeviceID, b.DeviceID)
	})
	writeJSON(w, http.StatusOK, devices)
}

func (s *Server) handleActiveAlarms(w http.ResponseWriter, r *http.Request) {
	alarms, err := s.store.ActiveAlarms(r.Context())
	if err != nil {
		log.Printf("active alarms: %v", err)
		writeError(w, http.StatusInternalServerError, "could not load active alarms")
		return
	}
	writeJSON(w, http.StatusOK, toAlarmsJSON(alarms))
}

func (s *Server) handleAlarms(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device")

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, http.StatusBadRequest, "limit must be a number between 1 and 500")
			return
		}
		limit = n
	}

	alarms, err := s.store.RecentAlarms(r.Context(), device, limit)
	if err != nil {
		log.Printf("recent alarms: %v", err)
		writeError(w, http.StatusInternalServerError, "could not load alarms")
		return
	}
	writeJSON(w, http.StatusOK, toAlarmsJSON(alarms))
}

// handleStream keeps the connection open and pushes events as they happen.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := s.hub.Subscribe()
	defer unsubscribe()

	fmt.Fprint(w, ": connected\n\n") // lines starting with ':' are comments
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done(): // browser closed the tab, or server is shutting down
			return
		case msg, open := <-ch:
			if !open {
				return
			}
			w.Write(msg)
			flusher.Flush()
		case <-heartbeat.C: // keeps proxies from closing an idle connection
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// ---------- Helpers ----------

// withCORS lets the React dev server (a different port) call this API.
// This API is read-only, so allowing any origin is acceptable for now.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
