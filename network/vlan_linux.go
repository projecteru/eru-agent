package network

import (
	"net"
	"os/exec"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/docker/libcontainer/netlink"
)

func AddVLan(vethName, ips, cid string) bool {
	lock.Lock()
	defer lock.Unlock()
	device, _ := Devices.Get(cid, 0)
	logs.Info("Add new VLan to", vethName, cid)

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}

	if err := netlink.NetworkLinkAddMacVlan(device, vethName, "bridge"); err != nil {
		logs.Info("Create macvlan device failed", err)
		return false
	}

	ifc, _ := net.InterfaceByName(vethName)
	if err := netlink.NetworkSetNsPid(ifc, container.State.Pid); err != nil {
		logs.Info("Set macvlan device into container failed", err)
		delVLan(vethName)
		return false
	}

	cmd := exec.Command("nsenter", "-t", container.State.Pid, "-n", "ip", "addr", "add", ips, "dev", vethName)
	if err := cmd.Run(); err != nil {
		logs.Info("Bind ip in container failed", err)
		return false
	}
	cmd = exec.Command("nsenter", "-t", container.State.Pid, "-n", "ip", "link", "set", vethName, "up")
	if err := cmd.Run(); err != nil {
		logs.Info("Set up veth in container failed", err)
		return false
	}
	logs.Info("Add VLAN device success", cid)
	return true
}

func SetDefaultRoute(cid, gateway string) bool {
	lock.Lock()
	defer lock.Unlock()
	logs.Info("Set", cid[:12], "default route", gateway)

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("RouteSetter inspect docker failed", err)
		return false
	}

	cmd := exec.Command("nsenter", "-t", container.State.Pid, "-n", "route", "del", "default")
	if err := cmd.Run(); err != nil {
		logs.Info("Clean default route failed", err)
		return false
	}

	cmd := exec.Command("nsenter", "-t", container.State.Pid, "-n", "route", "add", "default", "gw", gateway)
	if err := cmd.Run(); err != nil {
		logs.Info("RouteSetter set default route failed", err)
		return false
	}

	logs.Info("Set default route success", cid, gateway)
	return true
}
