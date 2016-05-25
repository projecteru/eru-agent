package network

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/utils"
	"github.com/vishvananda/netlink"
)

var Devices *utils.HashBackends
var lock sync.Mutex

func InitVlan() {
	Devices = utils.NewHashBackends(g.Config.VLan.Physical)
	log.Info("Vlan initiated")
}

func DelVlan(link netlink.Link) {
	if err := netlink.LinkDel(link); err != nil {
		log.Errorf("Delete device failed %s", err)
	}
}
