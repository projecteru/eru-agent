package network

import (
	"net"
	"runtime"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/projecteru/eru-agent/g"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
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
		log.Errorf("Get orignal namespace failed %s", err)
		return false
	}
	defer origins.Close()

	ns, err := netns.GetFromPid(pid)
	if err != nil {
		log.Errorf("Get container namespace failed %s", err)
		return false
	}

	netns.Set(ns)
	defer ns.Close()
	defer netns.Set(origins)

	if err := delDefaultRoute(); err != nil {
		log.Errorf("Delete default routing table failed %s", err)
		return false
	}

	if err := addDefaultRoute(gateway); err != nil {
		log.Errorf("Add default route failed %s", err)
		return false
	}

	log.Infof("Set default route success %s %s", cid[:12], gateway)
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
		log.Errorf("Get orignal namespace failed %s", err)
		return false
	}
	defer origins.Close()

	ns, err := netns.GetFromPid(pid)
	if err != nil {
		log.Errorf("Get container namespace failed %s", err)
		return false
	}

	netns.Set(ns)
	defer ns.Close()
	defer netns.Set(origins)

	if err := addRouteByLink(CIDR, ifc); err != nil {
		log.Errorf("Add route failed %s", err)
		return false
	}

	log.Infof("Add route success %s %s %s", cid[:12], CIDR, ifc)
	return true
}

func AddRoute(cid, CIDR, ifc string) bool {
	lock.Lock()
	defer lock.Unlock()

	log.Infof("Add %s route %s %s", cid[:12], CIDR, ifc)

	ctx := context.Background()
	container, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("RouteSetter inspect docker failed %s", err)
		return false
	}

	pid := container.State.Pid

	return addRoute(cid, CIDR, ifc, pid)
}

func SetDefaultRoute(cid, gateway string) bool {
	lock.Lock()
	defer lock.Unlock()

	log.Infof("Set %s default route %s", cid[:12], gateway)

	ctx := context.Background()
	container, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("RouteSetter inspect docker failed %s", err)
		return false
	}

	pid := container.State.Pid
	return setDefaultRoute(cid, gateway, pid)
}
