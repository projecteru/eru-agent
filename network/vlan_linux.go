package network

import (
	"net"
	"os/exec"
	"strconv"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/docker/libcontainer/netlink"
)

func AddVLan(vethName, ips, containerID string) bool {
	lock.Lock()
	defer lock.Unlock()
	device, _ := Devices.Get(containerID, 0)
	logs.Info("Add new VLan to", vethName, containerID)

	if err := netlink.NetworkLinkAddMacVlan(device, vethName, "bridge"); err != nil {
		logs.Info("Create macvlan device failed", err)
		return false
	}

	container, err := g.Docker.InspectContainer(containerID)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		delVLan(vethName)
		return false
	}

	ifc, _ := net.InterfaceByName(vethName)
	if err := netlink.NetworkSetNsPid(ifc, container.State.Pid); err != nil {
		logs.Info("Set macvlan device into container failed", err)
		delVLan(vethName)
		return false
	}

	pid := strconv.Itoa(container.State.Pid)
	cmd := exec.Command("nsenter", "-t", pid, "-n", "ip", "addr", "add", ips, "dev", vethName)
	if err := cmd.Run(); err != nil {
		logs.Info("Bind ip in container failed", err)
		return false
	}
	cmd = exec.Command("nsenter", "-t", pid, "-n", "ip", "link", "set", vethName, "up")
	if err := cmd.Run(); err != nil {
		logs.Info("Set up veth in container failed", err)
		return false
	}
	logs.Info("Add VLAN device success", containerID)
	return true
}
