package lenz

import (
	"os"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var Lenz *LenzForwarder

type LenzForwarder struct {
	Attacher *AttachManager
	Router   *RouteManager
	Routefs  RouteFileStore
}

func NewLenz() *LenzForwarder {
	obj := &LenzForwarder{}
	obj.Attacher = NewAttachManager(g.Docker)
	obj.Router = NewRouteManager(obj.Attacher, g.Config.Lenz.Stdout)
	obj.Routefs = RouteFileStore(g.Config.Lenz.Routes)

	if len(g.Config.Lenz.Forwards) > 0 {
		logs.Info("Routing all to", g.Config.Lenz.Forwards)
		target := defines.Target{Addrs: g.Config.Lenz.Forwards}
		route := defines.Route{ID: "lenz_default", Target: &target}
		route.LoadBackends()
		obj.Router.Add(&route)
	}

	if _, err := os.Stat(g.Config.Lenz.Routes); err == nil {
		logs.Info("Loading and persisting routes in", g.Config.Lenz.Routes)
		logs.Assert(obj.Router.Load(obj.Routefs), "persistor")
	}
	return obj
}
