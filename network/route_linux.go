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
		logs.Info("Add default route failed", err)
		return false
	}

	logs.Info("Set default route success", cid[:12], gateway)
	return true
}

func addRouteByLink(CIDR, ifc string) error {
	link, err := netlink.LinkByName(ifc)
	if err != nil {
		return err
	}
	_, dst, err := net.ParseCIDR(CIDR)
	if err != nil {
		return err
	}
	route := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst}
	return netlink.RouteAdd(&route)
}

func addRoute(cid, CIDR, ifc string, pid int) bool {
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

	if err := addRouteByLink(CIDR, ifc); err != nil {
		logs.Info("Add route failed", err)
		return false
	}

	logs.Info("Add route success", cid[:12], CIDR, ifc)
	return true
}

func AddRoute(cid, CIDR, ifc string) bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("Add", cid[:12], "route", CIDR, ifc)

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("RouteSetter inspect docker failed", err)
		return false
	}

	pid := container.State.Pid

	return addRoute(cid, CIDR, ifc, pid)
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
