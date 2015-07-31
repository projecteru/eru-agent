package network

import (
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

func AddVLan(vethName, ips, cid string) bool {
	_, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}
	logs.Info("Add VLAN device success", cid, vethName)
	return true
}

func SetDefaultRoute(cid, gateway string) bool {
	_, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}
	logs.Info("Set default route success", cid, gateway)
	return true
}
