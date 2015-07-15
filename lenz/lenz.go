package lenz

import (
	"os"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var Attacher *AttachManager
var Router *RouteManager
var Routefs RouteFileStore

func InitLenz() {
	Attacher = NewAttachManager(g.Docker)
	Router = NewRouteManager(Attacher, g.Config.Lenz.Stdout)
	Routefs = RouteFileStore(g.Config.Lenz.Routes)
	if len(g.Config.Lenz.Forwards) > 0 {
		logs.Info("Routing all to", g.Config.Lenz.Forwards)
		target := defines.Target{Addrs: g.Config.Lenz.Forwards}
		route := defines.Route{ID: "lenz_default", Target: &target}
		route.LoadBackends()
		Router.Add(&route)
	}
	if _, err := os.Stat(g.Config.Lenz.Routes); err == nil {
		logs.Info("Loading and persisting routes in", g.Config.Lenz.Routes)
		logs.Assert(Router.Load(Routefs), "persistor")
	}
}
