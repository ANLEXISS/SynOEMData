package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"synora/pkg/contract"
)

type baselineSnapshotStore struct{ body []byte }

func (s baselineSnapshotStore) SnapshotJSON() ([]byte, error) {
	return append([]byte(nil), s.body...), nil
}

type baselineSubscription struct{ events chan contract.Event }

func (s baselineSubscription) Subscribe(context.Context) <-chan contract.Event { return s.events }

type baselineResidentService struct{ view ResidentGalleryView }

func (s baselineResidentService) Create(context.Context, string, string) (ResidentGalleryView, bool, error) {
	return s.view, false, nil
}
func (s baselineResidentService) Status(context.Context, string) (ResidentGalleryView, error) {
	return s.view, nil
}
func (s baselineResidentService) Rollback(context.Context, string) (ResidentGalleryView, error) {
	return s.view, nil
}
func (s baselineResidentService) Delete(context.Context, string) (ResidentGalleryView, error) {
	return s.view, nil
}

func readAPIFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(body)
}
func assertJSONFixture(t *testing.T, got, want []byte) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal(got, &actual); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(expected)
	if !bytes.Equal(a, b) {
		t.Fatalf("API JSON differs: got %s want %s", a, b)
	}
}

func TestH1APIStateHealthWebsocketAndResidentResponses(t *testing.T) {
	state := readAPIFixture(t, "state-response-v1.json")
	api := &WebAPI{Boundary: &Boundary{Store: baselineSnapshotStore{body: state}}, Health: func() map[string]any { return map[string]any{"service": "discovery", "status": "ok", "version": "v1"} }}
	for _, test := range []struct{ path, fixture string }{{"/api/v1/state", "state-response-v1.json"}, {"/api/v1/health", "health-response-v1.json"}} {
		recorder := httptest.NewRecorder()
		api.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d", test.path, recorder.Code)
		}
		assertJSONFixture(t, recorder.Body.Bytes(), readAPIFixture(t, test.fixture))
	}
	event := contract.Event{ID: "event-h3", Type: "sensor.motion", Source: "sensor-h3", Timestamp: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC), Payload: map[string]any{"movement": true}}
	updates := make(chan contract.Event, 1)
	updates <- event
	close(updates)
	api.Updates = baselineSubscription{events: updates}
	recorder := httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/subscribe", nil))
	if recorder.Code != http.StatusOK || !strings.HasPrefix(recorder.Body.String(), "event: update\ndata: ") {
		t.Fatalf("SSE response=%q", recorder.Body.String())
	}
	line := strings.TrimSuffix(strings.TrimPrefix(recorder.Body.String(), "event: update\ndata: "), "\n\n")
	assertJSONFixture(t, []byte(line), readAPIFixture(t, "websocket-event-v1.json"))
	ref := "res_0123456789abcdef0123456789abcdef"
	api.Residents = baselineResidentService{view: ResidentGalleryView{ResidentRef: ref, GalleryStatus: "active", Generation: 3, PolicyVersion: "resident-gallery/v1"}}
	recorder = httptest.NewRecorder()
	api.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/residents/"+ref+"/face-gallery/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("resident status response=%d", recorder.Code)
	}
	assertJSONFixture(t, recorder.Body.Bytes(), readAPIFixture(t, "resident-gallery-response-v1.json"))
}
