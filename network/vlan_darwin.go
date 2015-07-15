package network

import "github.com/HunanTV/eru-agent/logs"

func AddVLan(vethName, ips, ident, containerID string) bool {
	logs.Info("Add VLAN device success", containerID, ident, vethName)
	return true
}
