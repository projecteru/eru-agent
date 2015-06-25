package main

import "./logs"

func (self *VLanSetter) addVLan(feedKey, content, ident, containerID string) {
	logs.Info("Add VLAN device success", containerID, ident)
}
