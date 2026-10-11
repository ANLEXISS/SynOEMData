package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultDeviceConfigPath = "/etc/synora/devices.yaml"

type KnownStation struct {
	DeviceID  string
	MAC       string
	StaticIP  string
	AllowWiFi bool
	Trust     string
}

func RenderAllowedStations(stations []KnownStation) string {
	macs := make([]string, 0, len(stations))
	for _, station := range stations {
		if station.AllowWiFi && station.Trust == "paired" && station.MAC != "" {
			macs = append(macs, station.MAC)
		}
	}
	sort.Strings(macs)
	return strings.Join(macs, "\n") + map[bool]string{true: "\n", false: ""}[len(macs) > 0]
}

func WriteAllowedStations(cfg SynoraNetConfig, stations []KnownStation) ([]KnownStation, error) {
	if !cfg.AccessControl.Enabled || !cfg.AccessControl.StationAllowlist {
		stations = nil
	}
	path := cfg.AccessControl.MACAllowlistFile
	if strings.TrimSpace(path) == "" {
		path = DefaultRunDir + "/hostapd-allowed-stations"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(RenderAllowedStations(stations)), 0600); err != nil {
		return nil, fmt.Errorf("write station allowlist: %w", err)
	}
	_ = os.Chmod(path, 0600)
	return stations, nil
}

func KnownStationCount(stations []KnownStation) int {
	count := 0
	for _, station := range stations {
		if station.AllowWiFi && station.Trust == "paired" {
			count++
		}
	}
	return count
}

func HasStationSecurityWarning(stations []KnownStation) bool {
	for _, station := range stations {
		if station.Trust != "paired" {
			return true
		}
	}
	return false
}
