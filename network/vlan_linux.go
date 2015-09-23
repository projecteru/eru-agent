package network

import (
	"runtime"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/krhubert/netns"
	"github.com/vishvananda/netlink"
)

func setUpVLan(cid, ips string, pid int, veth netlink.Link) bool {
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

	addr, err := netlink.ParseAddr(ips)
	if err != nil {
		logs.Info("Parse CIDR failed", err)
		return false
	}

	if err := netlink.AddrAdd(veth, addr); err != nil {
		logs.Info("Add addr to veth failed", err)
		return false
	}

	if err := netlink.LinkSetUp(veth); err != nil {
		logs.Info("Setup veth failed", err)
		return false
	}

	logs.Info("Add vlan device success", cid[:12])
	return true
}

func AddVLan(vethName, ips, cid string) bool {
	lock.Lock()
	defer lock.Unlock()
	device, _ := Devices.Get(cid, 0)
	logs.Info("Add new VLan to", vethName, cid[:12])

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}

	parent, err := netlink.LinkByName(device)
	if err != nil {
		logs.Info("Get parent NIC failed", err)
		return false
	}

	veth := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{Name: vethName, ParentIndex: parent.Attrs().Index},
		Mode:      netlink.MACVLAN_MODE_BRIDGE,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		logs.Info("Create macvlan device failed", err)
		return false
	}

	if err := netlink.LinkSetNsPid(veth, container.State.Pid); err != nil {
		logs.Info("Set macvlan device into container failed", err)
		delVLan(veth)
		return false
	}

	return setUpVLan(cid, ips, container.State.Pid, veth)
}
