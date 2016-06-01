package network

import (
	log "github.com/Sirupsen/logrus"
	"github.com/projecteru/eru-agent/g"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/context"
)

func AddVlan(vethName, ips, cid string) bool {
	ctx := context.Background()
	_, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("VLanSetter inspect docker failed %s", err)
		return false
	}
	log.Infof("Add VLAN device success %s %s", cid, vethName)
	return true
}

func DelMacVlanDevice(vethName string) error {
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

func AddPrerouting(eip, dest, ident string) error {
	return nil
}

func DelPrerouting(eip, dest, ident string) error {
	return nil
}

func SetBroadcast(vethName, ip string) error {
	return nil
}
