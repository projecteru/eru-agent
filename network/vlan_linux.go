package network

import (
	"os/exec"
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

	if err := BindAndSetup(veth, ips); err != nil {
		logs.Info("Bind and setup NIC failed", err)
		return false
	}

	logs.Info("Add vlan device success", cid[:12])
	return true
}

func AddVLan(vethName, ips, cid string) bool {
	lock.Lock()
	defer lock.Unlock()

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}

	if err := AddMacVlanDevice(vethName, cid); err != nil {
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

func AddMacVlanDevice(vethName, seq string) error {
	device, _ := Devices.Get(seq, 0)
	logs.Info("Add new macvlan device", vethName, device)

	parent, err := netlink.LinkByName(device)
	if err != nil {
		return err
	}

	veth := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{Name: vethName, ParentIndex: parent.Attrs().Index},
		Mode:      netlink.MACVLAN_MODE_BRIDGE,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return err
	}
	return nil
}

func BindAndSetup(veth netlink.Link, ips string) error {
	addr, err := netlink.ParseAddr(ips)
	if err != nil {
		return err
	}

	if err := netlink.AddrAdd(veth, addr); err != nil {
		return err
	}

	if err := netlink.LinkSetUp(veth); err != nil {
		return err
	}
	return nil
}

func AddCalico(env []string, multiple bool, cid, vethName, ip string) error {
	if !multiple {
		add := exec.Command("calicoctl", "container", "add", cid, ip, "--interface", vethName)
		add.Env = env
		return add.Run()
	}
	add := exec.Command("calicoctl", "container", cid, "ip", "add", ip, "--interface", vethName)
	add.Env = env
	return add.Run()
}

func BindCalicoProfile(env []string, cid, profile string) error {
	profile := exec.Command("calicoctl", "container", cid, "profile", "append", profile)
	profile.Env = env
	return profile.Run()
}
