package vision

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const baselineFixtureRoot = "../../testdata/contract-baseline/vision"

func baselineFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(baselineFixtureRoot, name))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(body)
}

func TestH1GoVisionRequestMatchesSharedFixture(t *testing.T) {
	request := Request{RequestID: "h1-clip-001", Operation: VisionClipProcess, ID: "clip-h1", ClipPath: "/tmp/synora-h1-fixture.mp4", CameraID: "camera-h1"}
	got, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(baselineFixture(t, "go-clip-request-v1.json"), &wantValue); err != nil {
		t.Fatal(err)
	}
	got, _ = json.Marshal(gotValue)
	want, _ := json.Marshal(wantValue)
	if !bytes.Equal(got, want) {
		t.Fatalf("Go request no longer matches shared Python input fixture:\n got %s\nwant %s", got, want)
	}
}

func TestH1GoConsumesPythonVisionResponseFixture(t *testing.T) {
	var response WorkerResponse
	if err := json.Unmarshal(baselineFixture(t, "python-worker-response-v1.json"), &response); err != nil {
		t.Fatal(err)
	}
	if response.RequestID != "h1-python-response-001" || len(response.Events) != 1 {
		t.Fatalf("response correlation/events changed: %#v", response)
	}
	if response.Events[0].Type != "vision.unknown" || response.Events[0].Source != "camera-h1" || response.Events[0].SceneID != "scene-h1" || response.Events[0].Timestamp != 1700000000.25 {
		t.Fatalf("Python event semantics changed at Go boundary: %#v", response.Events[0])
	}
	if got := response.Events[0].Payload["identity"]; got != "unknown" {
		t.Fatalf("Python payload identity was lost: %#v", response.Events[0].Payload)
	}
	if got := response.Events[0].Payload["scene"].(map[string]any)["fall"]; got != false {
		t.Fatalf("nested Python payload was lost: %#v", response.Events[0].Payload)
	}
	var source struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(baselineFixture(t, "python-worker-response-v1.json"), &source); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Events[0].Payload, source.Events[0]["payload"]) {
		t.Fatalf("Python payload changed during Go decode:\n got %#v\nwant %#v", response.Events[0].Payload, source.Events[0]["payload"])
	}
	// Structural decoding does not turn events into commands; unknown event names
	// remain inert data at this boundary and are not mapped to an action type.
	var extended map[string]any
	if err := json.Unmarshal(baselineFixture(t, "python-worker-response-v1.json"), &extended); err != nil {
		t.Fatal(err)
	}
	events := extended["events"].([]any)
	events[0].(map[string]any)["type"] = "device.command"
	unknown, _ := json.Marshal(extended)
	decoded, err := decodeWorkerResponse(unknown)
	if err != nil {
		t.Fatalf("current structural decoder unexpectedly rejected event name: %v", err)
	}
	if decoded.Events[0].Type != "device.command" {
		t.Fatalf("unknown event was transformed: %q", decoded.Events[0].Type)
	}
}

func TestH1VisionPayloadShapesAndDuplicateKeys(t *testing.T) {
	for _, raw := range []string{
		`{"request_id":"missing","events":[{"type":"vision.unknown","source":"camera-h1","timestamp":1}]}`,
		`{"request_id":"null","events":[{"type":"vision.unknown","source":"camera-h1","timestamp":1,"payload":null}]}`,
		string(baselineFixture(t, "python-worker-empty-payload-v1.json")),
	} {
		response, err := decodeWorkerResponse(json.RawMessage(raw))
		if err != nil {
			t.Fatalf("empty/missing payload rejected: %v", err)
		}
		if len(response.Events) != 1 || len(response.Events[0].Payload) != 0 {
			t.Fatalf("empty payload not preserved as empty: %#v", response)
		}
	}
	for _, payload := range []string{`[]`, `"text"`, `3`} {
		raw := `{"request_id":"bad-shape","events":[{"type":"vision.unknown","source":"camera-h1","timestamp":1,"payload":` + payload + `}]}`
		if _, err := decodeWorkerResponse(json.RawMessage(raw)); !errors.Is(err, ErrVisionMalformedResponse) {
			t.Errorf("payload shape %s error=%v", payload, err)
		}
	}
	duplicate := `{"request_id":"duplicate","events":[{"type":"vision.unknown","source":"camera-h1","timestamp":1,"payload":{"identity":"first","identity":"last"}}]}`
	response, err := decodeWorkerResponse(json.RawMessage(duplicate))
	if err != nil {
		t.Fatal(err)
	}
	if response.Events[0].Payload["identity"] != "last" {
		t.Fatalf("Go duplicate-key behavior changed: %#v", response.Events[0].Payload)
	}
	if _, err := decodeWorkerResponse(json.RawMessage(`{"request_id":"broken","events":[`)); !errors.Is(err, ErrVisionMalformedResponse) {
		t.Fatalf("malformed JSON error=%v", err)
	}
	if _, err := decodeWorkerResponse(json.RawMessage(`{"request_id":"oversize","events":[{"type":"vision.unknown","source":"camera-h1","timestamp":1,"payload":{"text":"` + strings.Repeat("x", 1<<20) + `"}}]}`)); err != nil {
		t.Fatalf("current protocol has no payload byte limit; 1MiB structured payload rejected: %v", err)
	}
}

func TestH1GoVisionInvalidResponseUsesCurrentProtocolFailure(t *testing.T) {
	for _, raw := range []string{`{`, `{"events":[]}`, `{"request_id":"","events":[]}`, `{"request_id":"x","events":null}`} {
		if _, err := decodeWorkerResponse(json.RawMessage(raw)); !errors.Is(err, ErrVisionMalformedResponse) {
			t.Errorf("decodeWorkerResponse(%q) error=%v, want ErrVisionMalformedResponse", raw, err)
		}
	}
}

func TestH3RuntimeProtocolPreservesPythonPayloadThroughWorkerSocket(t *testing.T) {
	fixture := baselineFixture(t, "python-worker-response-v1.json")
	server, path := startRuntimeProtocolServer(t, "normal", func(conn net.Conn) error {
		var request Request
		if err := json.NewDecoder(conn).Decode(&request); err != nil { return err }
	if request.RequestID != "h1-python-response-001" || request.Operation != VisionClipProcess { return fmt.Errorf("unexpected worker request %#v", request) }
		return writeFragmented(conn, string(fixture)+"\n")
	})
	defer server.close()
	manager := NewWorkerManager(nil, WorkerManagerConfig{Executor:&runtimeTestExecutor{}})
	runtime := NewRuntimeWithManagerAndSocketTimeout(manager,path,time.Second)
	response,err := runtime.Process(&ClipJob{ID:"h1-python-response-001",CameraID:"camera-h1",Path:"/tmp/synora-h1-fixture.mp4"})
	if err != nil { t.Fatal(err) }
	if len(response.Events)!=1 || response.Events[0].Payload["identity"]!="unknown" || response.Events[0].Payload["clip_path"]!="/tmp/synora-h1-fixture.mp4" {
		t.Fatalf("Python payload did not survive the actual worker socket: %#v",response)
	}
	_ = runtime.Close(); _ = manager.Stop("camera-h1"); server.wait(t)
}
