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
func addDefaultRoute(gateway string) error {

	gwIP := net.ParseIP(gateway)
	route := netlink.Route{Gw: gwIP}

	return netlink.RouteAdd(&route)
}

//delete default routes
//FIXME all default routes will be erased
func delDefaultRoute() error {
	routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)

	for _, route := range routes {
		if route.Dst != nil || route.Src != nil {
			continue
		}
		if err := netlink.RouteDel(&route); err != nil {
			return err
		}
	}
	return nil
}

func setDefaultRoute(cid, gateway string, pid int) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origins, err := netns.Get()
	if err != nil {
		logs.Info("Get orignal namespace failed", err)
		return false
	}
	defer origins.Close()

	ns, err := netns.GetFromPid(pid)
	if err != nil {
		logs.Info("Get container namespace failed", err)
		return false
	}

	netns.Set(ns)
	defer ns.Close()
	defer netns.Set(origins)

	if err := delDefaultRoute(); err != nil {
		logs.Info("Delete default routing table failed", err)
		return false
	}

	if err := addDefaultRoute(gateway); err != nil {
		logs.Info("add default routing table failed", err)
		return false
	}

	logs.Info("Set default route success", cid[:12], gateway)
	return true
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

	return setDefaultRoute(cid, gateway, pid)
}
