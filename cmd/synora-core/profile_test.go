package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"synora/internal/bus"
	"synora/internal/cognitivecore"
	"synora/pkg/contract"
)

// This reduced runtime exercise composes Foundation Bus and Intelligence Core
// without Discovery, Vision, Automation, or Platform services.
func TestCoreOnlyCompositionProcessesBusEvent(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "bus.sock")
	server := bus.NewServerWithConfig(socket, bus.ServerConfig{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		_ = server.Start()
	}()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case <-serverDone:
		case <-time.After(time.Second):
			t.Error("bus server did not stop")
		}
	})
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bus socket did not become ready")
		}
		time.Sleep(time.Millisecond)
	}

	client, err := bus.NewClientWithConfig(socket, "core", bus.ClientConfig{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	state := cognitivecore.NewUniversalStore()
	core := &cognitivecore.Core{
		Store: state,
		MLP:   cognitivecore.UnavailableMLP{Reason: "profile smoke test has no model bundle"},
		Gate:  cognitivecore.SafetyGate{DryRun: true},
	}
	service := &cognitivecore.Service{Bus: client, Core: core, Name: "core"}
	ctx, cancel := context.WithCancel(context.Background())
	serviceDone := make(chan error, 1)
	go func() { serviceDone <- service.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-serviceDone:
		case <-time.After(time.Second):
			t.Error("Core service did not stop")
		}
	})

	payload, err := json.Marshal(map[string]any{"device_id": "camera-profile", "reason": "offline"})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Send(contract.Message{
		ID: "profile-event-1", Type: contract.EventDeviceOffline,
		Kind: contract.KindEvent, Source: "core", Target: "core",
		Timestamp: time.Now().UTC(), Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for len(state.Journal()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := len(state.Journal()); got != 1 {
		t.Fatalf("Core journal commits=%d, want one event in reduced profile", got)
	}
}
