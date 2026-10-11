package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNetworkStationSourceFiltersAndNormalizesDeviceConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.yaml")
	data := []byte("devices:\n  - id: camera-b\n    type: camera\n    enabled: true\n    network:\n      mac: 'AA:BB:CC:DD:EE:02'\n      allow_wifi: 'true'\n      network_trust: paired\n      static_ip: 10.77.0.3\n  - id: camera-disabled\n    type: camera\n    enabled: false\n    network:\n      mac: 'AA:BB:CC:DD:EE:03'\n  - id: sensor-a\n    type: sensor\n    enabled: true\n    network:\n      mac: 'AA:BB:CC:DD:EE:04'\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	stations, err := (networkStationSource{}).LoadStations(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 1 {
		t.Fatalf("stations=%#v, want one active camera", stations)
	}
	got := stations[0]
	if got.DeviceID != "camera-b" || got.MAC != "aa:bb:cc:dd:ee:02" || got.StaticIP != "10.77.0.3" || !got.AllowWiFi || got.Trust != "paired" {
		t.Fatalf("station=%#v", got)
	}
}
