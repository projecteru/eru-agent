package network

import (
	"os/exec"
	"runtime"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/projecteru/eru-agent/g"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func setUpVLan(cid, ips string, pid int, veth netlink.Link) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		log.Errorf("Get orignal namespace failed %s", err)
		return false
	}
	defer origns.Close()

	ns, err := netns.GetFromPid(pid)
	if err != nil {
		log.Errorf("Get container namespace failed %s", err)
		return false
	}

	netns.Set(ns)
	defer ns.Close()
	defer netns.Set(origns)

	if err := BindAndSetup(veth, ips); err != nil {
		log.Errorf("Bind and setup NIC failed %s", err)
		DelVlan(veth)
		return false
	}

	log.Infof("Add vlan device success %s", cid[:12])
	return true
}

func AddVlan(vethName, ips, cid string) bool {
	lock.Lock()
	defer lock.Unlock()

	ctx := context.Background()
	container, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("VLanSetter inspect docker failed %s", err)
		return false
	}

	veth, err := AddMacVlanDevice(vethName, cid)
	if err != nil {
		log.Errorf("Create macvlan device failed %s", err)
		return false
	}

	if err := netlink.LinkSetNsPid(veth, container.State.Pid); err != nil {
		log.Errorf("Set macvlan device into container failed %s", err)
		DelVlan(veth)
		return false
	}

	return setUpVLan(cid, ips, container.State.Pid, veth)
}

func DelMacVlanDevice(vethName string) error {
	log.Infof("Release macvlan device %s", vethName)
	link, err := netlink.LinkByName(vethName)
	if err != nil {
		return err
	}
	DelVlan(link)
	return nil
}

func AddMacVlanDevice(vethName, seq string) (netlink.Link, error) {
	device := Devices.Get(seq, 0)
	log.Info("Add new macvlan device %s %s", vethName, device)

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
