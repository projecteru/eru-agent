package network

import (
	"sync"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/vishvananda/netlink"
)

var Devices *consistent.Consistent
var lock sync.Mutex

func InitVlan() {
	Devices = consistent.New()
	for _, device := range g.Config.VLan.Physical {
		Devices.Add(device)
	}
	logs.Info("Vlan initiated")
}

func DelVlan(link netlink.Link) {
	if err := netlink.LinkDel(link); err != nil {
		logs.Debug("Delete device failed", err)
	}
}
