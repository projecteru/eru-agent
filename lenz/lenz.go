package lenz

import (
	"os"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var Attacher *AttachManager
var Router *RouteManager
var Routefs RouteFileStore

func InitLenz() {
	Attacher = NewAttachManager()
	Router = NewRouteManager(Attacher)
	Routefs = RouteFileStore(g.Config.Lenz.Routes)
	if len(g.Config.Lenz.Forwards) > 0 {
		logs.Debug("Lenz Routing all to", g.Config.Lenz.Forwards)
		target := defines.Target{Addrs: g.Config.Lenz.Forwards}
		route := defines.Route{ID: common.LENZ_DEFAULT, Target: &target}
		route.LoadBackends()
		Router.Add(&route)
	}
	if _, err := os.Stat(g.Config.Lenz.Routes); err == nil {
		logs.Debug("Loading and persisting routes in", g.Config.Lenz.Routes)
		logs.Assert(Router.Load(Routefs), "persistor")
	}
	logs.Info("Lenz initiated")
}

func CloseLenz() {
	logs.Info("Close all lenz streamer")
	routes, err := Router.GetAll()
	if err != nil {
		logs.Info("Get all lenz route failed", err)
		return
	}
	for _, route := range routes {
		if !Router.Remove(route.ID) {
			logs.Info("Close lenz route failed", route.ID)
		}
	}
}
