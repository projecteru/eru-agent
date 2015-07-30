package network

import "github.com/HunanTV/eru-agent/logs"

func AddVLan(vethName, ips, cid string) bool {
	logs.Info("Add VLAN device success", cid, vethName)
	return true
}

func SetDefaultRoute(cid, gateway string) bool {
	logs.Info("Set default route success", cid, gateway)
	return true
}
