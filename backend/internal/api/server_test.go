package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/grid"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/protection"
	"github.com/rohanshresthaaaa/gridsentinel/backend/internal/store"
)

// fakeStore records what it was asked for and returns canned data.
type fakeStore struct {
	alarms    []store.Alarm
	err       error
	gotDevice string
	gotLimit  int
}

func (f *fakeStore) ActiveAlarms(ctx context.Context) ([]store.Alarm, error) {
	return f.alarms, f.err
}

func (f *fakeStore) RecentAlarms(ctx context.Context, deviceID string, limit int) ([]store.Alarm, error) {
	f.gotDevice, f.gotLimit = deviceID, limit
	return f.alarms, f.err
}

func get(srv *Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealth(t *testing.T) {
	srv := NewServer(&fakeStore{}, NewHub())
	rec := get(srv, "/api/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("CORS header missing")
	}
}

func TestDevicesReturnsLatestReadingPerDevice(t *testing.T) {
	srv := NewServer(&fakeStore{}, NewHub())
	srv.PublishReading(grid.Reading{DeviceID: "RELAY-02", FrequencyHz: 60})
	srv.PublishReading(grid.Reading{DeviceID: "RELAY-01", FrequencyHz: 60})
	srv.PublishReading(grid.Reading{DeviceID: "RELAY-01", FrequencyHz: 59.2}) // newer

	var devices []readingJSON
	if err := json.Unmarshal(get(srv, "/api/devices").Body.Bytes(), &devices); err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("want 2 devices, got %d", len(devices))
	}
	if devices[0].DeviceID != "RELAY-01" || devices[0].FrequencyHz != 59.2 {
		t.Fatalf("want newest RELAY-01 reading first, got %+v", devices[0])
	}
}

func TestEmptyListsAreArraysNotNull(t *testing.T) {
	srv := NewServer(&fakeStore{}, NewHub())
	for _, path := range []string{"/api/devices", "/api/alarms/active", "/api/alarms"} {
		if body := strings.TrimSpace(get(srv, path).Body.String()); body != "[]" {
			t.Errorf("%s: got %s, want []", path, body)
		}
	}
}

func TestAlarmsPassesFiltersToStore(t *testing.T) {
	fake := &fakeStore{}
	srv := NewServer(fake, NewHub())

	get(srv, "/api/alarms?device=RELAY-02&limit=5")
	if fake.gotDevice != "RELAY-02" || fake.gotLimit != 5 {
		t.Fatalf("store got device=%q limit=%d", fake.gotDevice, fake.gotLimit)
	}

	get(srv, "/api/alarms")
	if fake.gotDevice != "" || fake.gotLimit != 50 {
		t.Fatalf("defaults wrong: device=%q limit=%d", fake.gotDevice, fake.gotLimit)
	}
}

func TestAlarmsRejectsBadLimit(t *testing.T) {
	srv := NewServer(&fakeStore{}, NewHub())
	for _, limit := range []string{"abc", "0", "-3", "501"} {
		rec := get(srv, "/api/alarms?limit="+limit)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%s: status %d, want 400", limit, rec.Code)
		}
	}
}

func TestStoreErrorReturns500WithoutLeakingDetails(t *testing.T) {
	srv := NewServer(&fakeStore{err: errors.New("secret connection string")}, NewHub())
	rec := get(srv, "/api/alarms/active")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("internal error details leaked to the client")
	}
}

func TestActiveAlarmsMarksActiveFlag(t *testing.T) {
	ended := time.Now()
	srv := NewServer(&fakeStore{alarms: []store.Alarm{
		{ID: 1, Element: "27"},
		{ID: 2, Element: "50", DroppedOutAt: &ended},
	}}, NewHub())

	var alarms []alarmJSON
	if err := json.Unmarshal(get(srv, "/api/alarms/active").Body.Bytes(), &alarms); err != nil {
		t.Fatal(err)
	}
	if !alarms[0].Active || alarms[1].Active {
		t.Fatalf("active flags wrong: %+v", alarms)
	}
}

func TestStreamDeliversPublishedEvents(t *testing.T) {
	hub := NewHub()
	srv := NewServer(&fakeStore{}, hub)
	ts := httptest.NewServer(srv.Handler()) // a real server on a random port
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}

	for hub.Count() == 0 { // wait until the stream has subscribed
		if ctx.Err() != nil {
			t.Fatal("stream never subscribed")
		}
		time.Sleep(10 * time.Millisecond)
	}

	srv.PublishTransition(protection.Transition{
		Kind: protection.Pickup,
		Event: protection.Event{
			DeviceID: "RELAY-03", Element: "50", Name: "Overcurrent", Phase: "A",
			Value: 300, Threshold: 100, Unit: "A", Timestamp: time.Now(),
		},
	})

	scanner := bufio.NewScanner(resp.Body)
	sawEvent := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "event: alarm" {
			sawEvent = true
			continue
		}
		if sawEvent && strings.HasPrefix(line, "data: ") {
			if !strings.Contains(line, `"kind":"PICKUP"`) || !strings.Contains(line, `"element":"50"`) {
				t.Fatalf("unexpected data line: %s", line)
			}
			return // success
		}
	}
	t.Fatalf("stream ended before the alarm arrived: %v", scanner.Err())
}
