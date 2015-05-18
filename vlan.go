package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"./common"
	"./logs"
	"github.com/CMGS/consistent"
	"github.com/docker/libcontainer/netlink"
	"github.com/keimoon/gore"
)

type VLanSetter struct {
	Devices *consistent.Consistent
}

func NewVLanSetter() *VLanSetter {
	v := &VLanSetter{}
	v.Devices = consistent.New()
	for _, device := range config.VLan.Physical {
		v.Devices.Add(device)
	}
	return v
}

func (self *VLanSetter) Watcher() {
	conn, err := common.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer common.Rds.Release(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:vlan", config.HostName)
	logs.Debug("Watch VLan Config", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			logs.Info("VLan Watcher Shutdown")
			break
		}
		command := string(message.Message)
		logs.Debug("Add new VLan", command)
		parser := strings.Split(command, "|")
		if len(parser) <= 3 {
			logs.Info("Command Invaild", command)
			continue
		}
		taskID, containerID, ident := parser[0], parser[1], parser[2]
		feedKey := fmt.Sprintf("eru:agent:%s:feedback", taskID)
		for _, content := range parser[3:] {
			self.addVLan(feedKey, content, ident, containerID)
		}
	}
}

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

	// Add macvlan device
	device, _ := self.Devices.Get(ident, 0)
	vethName := fmt.Sprintf("%s%s.%s", common.VLAN_PREFIX, ident, seq)
	logs.Info("Add new VLan to", vethName, containerID)

	// Create device
	if err := netlink.NetworkLinkAddMacVlan(device, vethName, "bridge"); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Create macvlan device failed", err)
		return
	}

	// Get Pid
	container, err := common.Docker.InspectContainer(containerID)
	if err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("VLanSetter inspect docker failed", err)
		self.delVLan(vethName)
		return
	}
	pid := strconv.Itoa(container.State.Pid)

	// Set into container
	ifc, _ := net.InterfaceByName(vethName)
	if err := netlink.NetworkSetNsPid(ifc, pid); err != nil {
		gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(conn)
		logs.Info("Set macvlan device into container failed", err)
		self.delVLan(vethName)
		return
	}

	cmd = exec.Command("nsenter", "-t", pid, "-n", "ip", "addr", "add", ips, "dev", vethName)
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

func (self *VLanSetter) delVLan(vethName string) {
	if err := netlink.NetworkLinkDel(vethName); err != nil {
		logs.Debug("Delete device failed", err)
	}
}
