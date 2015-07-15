package network

import "github.com/HunanTV/eru-agent/logs"

func addVLan(content, ident, containerID string) bool {
	logs.Info("Add VLAN device success", containerID, ident)
	return true
}
