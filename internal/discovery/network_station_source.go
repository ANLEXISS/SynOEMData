package discovery

import (
	"sort"
	"strconv"
	"strings"

	"synora/internal/device"
	"synora/internal/discovery/network"
	"synora/pkg/contract"
)

// networkStationSource translates the product device registry into the small
// host-network input type. The network adapter owns no device-file format or
// device business rules.
type networkStationSource struct{}

func (networkStationSource) LoadStations(path string) ([]network.KnownStation, error) {
	if strings.TrimSpace(path) == "" {
		path = network.DefaultDeviceConfigPath
	}
	items, err := device.Load(path)
	if err != nil {
		return nil, err
	}
	stations := make([]network.KnownStation, 0, len(items))
	for _, item := range items {
		if item.Type != contract.DeviceTypeCamera || !item.Enabled || item.DeletedAt != nil {
			continue
		}
		mac := network.NormalizeMAC(stationString(item.Network, "mac"))
		if mac == "" {
			continue
		}
		allow := stationBool(item.Network, "allow_wifi", true)
		trust := stationString(item.Network, "network_trust")
		if trust == "" {
			trust = "paired"
		}
		if trust != "paired" {
			allow = false
		}
		stations = append(stations, network.KnownStation{
			DeviceID: item.ID, MAC: mac,
			StaticIP:  firstStationValue(stationString(item.Network, "static_ip"), stationString(item.Network, "ip")),
			AllowWiFi: allow, Trust: trust,
		})
	}
	sort.Slice(stations, func(i, j int) bool { return stations[i].MAC < stations[j].MAC })
	return stations, nil
}

func stationString(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func stationBool(values map[string]any, key string, fallback bool) bool {
	switch value := values[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func firstStationValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
