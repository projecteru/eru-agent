package network

import (
	"net"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/docker/libcontainer/netlink"
	"github.com/krhubert/netns"
)

func setUpVLan(cid, vethName, ips string, pid int) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		logs.Info("Get orignal namespace failed", err)
		return false
	}
	defer origns.Close()

	ns, err := netns.GetFromPid(pid)
	if err != nil {
		logs.Info("Get container namespace failed", err)
		return false
	}

	netns.Set(ns)
	defer ns.Close()
	defer netns.Set(origns)

	ip, ipNet, err := net.ParseCIDR(ips)
	if err != nil {
		logs.Info("Parse CIDR failed", err)
		return false
	}

	ifc, err := net.InterfaceByName(vethName)
	if err != nil {
		logs.Info("Get container vlan failed", err)
		return false
	}

	netlink.NetworkLinkAddIp(ifc, ip, ipNet)
	netlink.NetworkLinkUp(ifc)
	logs.Info("Add vlan device success", cid)
	return true
}

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

	return setUpVLan(cid, vethName, ips, container.State.Pid)
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

	pid := strconv.Itoa(container.State.Pid)
	cmd := exec.Command("nsenter", "-t", pid, "-n", "route", "del", "default")
	if err := cmd.Run(); err != nil {
		logs.Info("Clean default route failed", err)
		return false
	}

	cmd = exec.Command("nsenter", "-t", pid, "-n", "route", "add", "default", "gw", gateway)
	if err := cmd.Run(); err != nil {
		logs.Info("RouteSetter set default route failed", err)
		return false
	}

	logs.Info("Set default route success", cid[:12], gateway)
	return true
}
