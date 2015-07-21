package network

import (
	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/docker/libcontainer/netlink"
)

var Devices *consistent.Consistent

func InitVlan() {
	Devices = consistent.New()
	for _, device := range g.Config.VLan.Physical {
		Devices.Add(device)
	}
	logs.Info("Vlan initiated")
}

func delVLan(vethName string) {
	if err := netlink.NetworkLinkDel(vethName); err != nil {
		logs.Debug("Delete device failed", err)
	}
}
