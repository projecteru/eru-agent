package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/logs"

	"github.com/docker/libcontainer/netlink"
	"github.com/keimoon/gore"
)

func (self *VLanSetter) addVLan(feedKey, content, ident, containerID string) {
	conn, err := common.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Info(err, "Get redis conn")
		return
	}
	defer common.Rds.Release(conn)

	parser := strings.Split(content, ":")
	if len(parser) != 2 {
		logs.Info("Seq and Ips Invaild", content)
		return
	}
	seq, ips := parser[0], parser[1]

	device, _ := self.Devices.Get(ident, 0)
	vethName := fmt.Sprintf("%s%s", common.VLAN_PREFIX, seq)
	logs.Info("Add new VLan to", vethName, containerID)

	if err := netlink.NetworkLinkAddMacVlan(device, vethName, "bridge"); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Create macvlan device failed", err)
		return
	}

	container, err := common.Docker.InspectContainer(containerID)
	if err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("VLanSetter inspect docker failed", err)
		self.delVLan(vethName)
		return
	}

	ifc, _ := net.InterfaceByName(vethName)
	if err := netlink.NetworkSetNsPid(ifc, container.State.Pid); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Set macvlan device into container failed", err)
		self.delVLan(vethName)
		return
	}

	pid := strconv.Itoa(container.State.Pid)
	cmd := exec.Command("nsenter", "-t", pid, "-n", "ip", "addr", "add", ips, "dev", vethName)
	if err := cmd.Run(); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Bind ip in container failed", err)
		return
	}
	cmd = exec.Command("nsenter", "-t", pid, "-n", "ip", "link", "set", vethName, "up")
	if err := cmd.Run(); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Set up veth in container failed", err)
		return
	}
	gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("1|%s|%s|%s", containerID, vethName, ips)).Run(conn)
	logs.Info("Add VLAN device success", containerID, ident)
}
