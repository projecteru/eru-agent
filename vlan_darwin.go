package main

import "github.com/HunanTV/eru-agent/logs"

func (self *VLanSetter) addVLan(feedKey, content, ident, containerID string) {
	logs.Info("Add VLAN device success", containerID, ident)
}
