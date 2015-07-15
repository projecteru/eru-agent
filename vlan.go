package main

import (
	"fmt"
	"strings"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/docker/libcontainer/netlink"
	"github.com/keimoon/gore"
)

type VLanSetter struct {
	Devices *consistent.Consistent
}

func NewVLanSetter() *VLanSetter {
	v := &VLanSetter{}
	v.Devices = consistent.New()
	for _, device := range g.Config.VLan.Physical {
		v.Devices.Add(device)
	}
	return v
}

func (self *VLanSetter) Watcher() {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer g.Rds.Release(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:vlan", g.Config.HostName)
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

func (self *VLanSetter) delVLan(vethName string) {
	if err := netlink.NetworkLinkDel(vethName); err != nil {
		logs.Debug("Delete device failed", err)
	}
}
