package network

import (
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/vishvananda/netlink"
)

func AddVlan(vethName, ips, cid string) bool {
	_, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}
	logs.Info("Add VLAN device success", cid, vethName)
	return true
}

func AddMacVlanDevice(vethName, seq string) error {
	return nil
}

func BindAndSetup(veth netlink.Link, ips string) error {
	return nil
}

func AddCalico(multiple bool, cid, vethName, ip string) error {
	return nil
}

func BindCalicoProfile(env []string, cid, profile string) error {
	return nil
}
