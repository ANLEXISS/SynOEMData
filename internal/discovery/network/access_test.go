package network

import (
	"os"
	"path/filepath"
	"testing"
)

type fakeStationSource struct {
	path  string
	items []KnownStation
}

func (f *fakeStationSource) LoadStations(path string) ([]KnownStation, error) {
	f.path = path
	return f.items, nil
}

func TestManagerUsesInjectedStationSource(t *testing.T) {
	allowlistPath := filepath.Join(t.TempDir(), "allowed-stations")
	source := &fakeStationSource{items: []KnownStation{{DeviceID: "cam-1", MAC: "aa:bb:cc:dd:ee:ff", AllowWiFi: true, Trust: "paired"}}}
	manager := NewManager(source)
	stations, err := manager.writeAllowedStations(SynoraNetConfig{AccessControl: AccessControlConfig{
		Enabled: true, StationAllowlist: true, MACAllowlistFile: allowlistPath,
	}}, "devices.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if source.path != "devices.yaml" || len(stations) != 1 {
		t.Fatalf("source path=%q stations=%#v", source.path, stations)
	}
	got, err := os.ReadFile(allowlistPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "aa:bb:cc:dd:ee:ff\n" {
		t.Fatalf("allowlist=%q", got)
	}
}
