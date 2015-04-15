package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"./common"
	"./logs"
	"github.com/CMGS/consistent"
	"github.com/keimoon/gore"
)

type VlanSetter struct {
	Devices *consistent.Consistent
}

func NewVlanSetter() *VlanSetter {
	v := &VlanSetter{}
	v.Devices = consistent.New()
	for _, device := range config.Vlan.Physical {
		v.Devices.Add(device)
	}
	return v
}

func (self *VlanSetter) Watcher() {
	conn, err := common.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer common.Rds.Release(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:vlan", config.HostName)
	logs.Debug("Watch Vlan Config", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			logs.Info("VLan Watcher Shutdown")
			break
		}
		command := string(message.Message)
		logs.Debug("Add new Vlan", command)
		parser := strings.Split(command, "|")
		containerID, ident := parser[0], parser[1]
		for _, seq := range parser[2:] {
			self.addVlan(seq, ident, containerID)
		}
	}
}

func (self *VlanSetter) addVlan(seq, ident, containerID string) {
	// Add macvlan device
	// TODO report err
	device, _ := self.Devices.Get(ident, 0)
	vethName := fmt.Sprintf("%s%s.%s", common.VLAN_PREFIX, ident, seq)
	logs.Info("Add new Vlan to", vethName, containerID)
	cmd := exec.Command("ip", "link", "add", vethName, "link", device, "type", "macvlan", "mode", "bridge")
	if err := cmd.Run(); err != nil {
		//TODO report to core
		logs.Info("Create macvlan device failed", err)
		return
	}
	container, err := common.Docker.InspectContainer(containerID)
	if err != nil {
		logs.Info("VlanSetter inspect docker failed", err)
		return
	}
	cmd = exec.Command("ip", "link", "set", "netns", strconv.Itoa(container.State.Pid), vethName)
	if err := cmd.Run(); err != nil {
		//TODO report to core
		logs.Info("Set macvlan device into container failed", err)
		return
	}
	//TODO report to core
	//TODO mission complete
	logs.Info("Add VLAN device success", containerID, ident)
}
