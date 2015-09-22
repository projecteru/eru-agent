package network

import (
	"net"
	"runtime"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/krhubert/netns"
	"github.com/vishvananda/netlink"
)

//add default route
func addDefaultRoute(gateway string) (err error) {

	gwIP := net.ParseIP(gateway)
	route := netlink.Route{Gw: gwIP}

	err = netlink.RouteAdd(&route)
	if err != nil {
		return err
	}
	return err
}

//delete default route
func delDefaultRoute() (err error) {
	routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)

	err = netlink.RouteDel(&routes[0])
	if err != nil {
		return err
	}
	return err
}

func setDefaultRoute(cpid int, gateway string) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origins, err := netns.Get()
	if err != nil {
		logs.Info("fail to get original namespace")
		return err
	}
	defer origins.Close()

	ns, _ := netns.GetFromPid(cpid)
	defer ns.Close()

	netns.Set(ns)
	defer netns.Set(origins)

	err = delDefaultRoute()
	if err != nil {
		logs.Info("delete default routing table failed")
		return err
	}

	err = addDefaultRoute(gateway)
	if err != nil {
		logs.Info("add default routing table failed")
		return err
	}

	return err
}

func SetDefaultRoute(cid, gateway string) bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("Set", cid[:12], "default route", gateway)

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("RouteSetter inspect docker failed", err)
		return false
	}

	pid := container.State.Pid

	err = setDefaultRoute(pid, gateway)
	if err != nil {
		logs.Info("set default route failed", err)
		return false
	}

	logs.Info("Set default route success", cid[:12], gateway)
	return true
}
