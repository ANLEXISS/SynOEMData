package network

import (
	"fmt"
)

const (
	BridgeName = "synorabr0"
	BridgeCIDR = "10.77.0.1/24"
)

func EnsureBridge() error { return ensureBridge(DefaultConfig().SynoraNet) }

func ensureBridge(cfg SynoraNetConfig) error {
	return ensureBridgeWithExecutor(cfg, OSCommandExecutor{})
}

func ensureBridgeWithExecutor(cfg SynoraNetConfig, commands HostCommandExecutor) error {
	commands = defaultHostCommandExecutor(commands)
	if cfg.GatewayIP == "" {
		cfg.GatewayIP = DefaultGateway
	}
	exists := commands.Run("ip", "link", "show", BridgeName) == nil
	if !exists {
		if err := commands.Run("ip", "link", "add", "name", BridgeName, "type", "bridge"); err != nil {
			return fmt.Errorf("create bridge failed: %w", err)
		}
	}
	_ = commands.Run("ip", "addr", "flush", "dev", BridgeName)
	if err := commands.Run("ip", "addr", "add", cfg.GatewayIP+"/24", "dev", BridgeName); err != nil {
		return fmt.Errorf("bridge ip: %w", err)
	}
	if err := commands.Run("ip", "link", "set", BridgeName, "up"); err != nil {
		return fmt.Errorf("bridge up: %w", err)
	}
	return nil
}
