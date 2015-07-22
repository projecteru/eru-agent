package network

import "github.com/HunanTV/eru-agent/logs"

func AddVLan(vethName, ips, containerID string) bool {
	lock.Lock()
	defer lock.Unlock()
	logs.Info("Add VLAN device success", containerID, vethName)
	return true
}
