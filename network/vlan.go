package network

import (
	"sync"

	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-agent/utils"
	"github.com/vishvananda/netlink"
)

var Devices *utils.HashBackends
var lock sync.Mutex

func InitVlan() {
	Devices = utils.NewHashBackends(g.Config.VLan.Physical)
	logs.Info("Vlan initiated")
}

func DelVlan(link netlink.Link) {
	if err := netlink.LinkDel(link); err != nil {
		logs.Debug("Delete device failed", err)
	}
}
