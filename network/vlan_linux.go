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
		DelVlan(veth)
		return false
	}

	logs.Info("Add vlan device success", cid[:12])
	return true
}

func AddVlan(vethName, ips, cid string) bool {
	lock.Lock()
	defer lock.Unlock()

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("VLanSetter inspect docker failed", err)
		return false
	}

	veth, err := AddMacVlanDevice(vethName, cid)
	if err != nil {
		logs.Info("Create macvlan device failed", err)
		return false
	}

	if err := netlink.LinkSetNsPid(veth, container.State.Pid); err != nil {
		logs.Info("Set macvlan device into container failed", err)
		DelVlan(veth)
		return false
	}

	return setUpVLan(cid, ips, container.State.Pid, veth)
}

func DelMacVlanDevice(vethName string) error {
	logs.Info("Release macvlan device", vethName)
	link, err := netlink.LinkByName(vethName)
	if err != nil {
		return err
	}
	DelVlan(link)
	return nil
}

func AddMacVlanDevice(vethName, seq string) (netlink.Link, error) {
	device, _ := Devices.Get(seq, 0)
	logs.Info("Add new macvlan device", vethName, device)

	parent, err := netlink.LinkByName(device)
	if err != nil {
		return nil, err
	}

	veth := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{Name: vethName, ParentIndex: parent.Attrs().Index},
		Mode:      netlink.MACVLAN_MODE_BRIDGE,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}
	return veth, nil
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

func SetBroadcast(vethName, ip string) error {
	cmd := exec.Command("ifconfig", vethName, "broadcast", ip)
	return cmd.Run()
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

func BindCalicoProfile(env []string, cid, profileName string) error {
	profile := exec.Command("calicoctl", "container", cid, "profile", "append", profileName)
	profile.Env = env
	return profile.Run()
}

func AddPrerouting(eip, dest, ident string) error {
	cmd := exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-d", eip, "-j", "DNAT", "--to-destination", dest, "-m", "comment", "--comment", ident)
	return cmd.Run()
}

func DelPrerouting(eip, dest, ident string) error {
	cmd := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-d", eip, "-j", "DNAT", "--to-destination", dest, "-m", "comment", "--comment", ident)
	return cmd.Run()
}
