package lenz

import (
	"os"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"

	log "github.com/Sirupsen/logrus"
)

var Attacher *AttachManager
var Router *RouteManager
var Routefs RouteFileStore

func InitLenz() {
	Attacher = NewAttachManager()
	Router = NewRouteManager(Attacher)
	Routefs = RouteFileStore(g.Config.Lenz.Routes)
	if len(g.Config.Lenz.Forwards) > 0 {
		log.Debugf("Lenz Routing all to %s", g.Config.Lenz.Forwards)
		target := defines.Target{Addrs: g.Config.Lenz.Forwards}
		route := defines.Route{ID: common.LENZ_DEFAULT, Target: &target}
		route.LoadBackends()
		Router.Add(&route)
	}
	if _, err := os.Stat(g.Config.Lenz.Routes); err == nil {
		log.Debugf("Loading and persisting routes in %s", g.Config.Lenz.Routes)
		log.Panicf("Persistor load error %s", Router.Load(Routefs))
	}
	log.Info("Lenz initiated")
}

func CloseLenz() {
	log.Info("Close all lenz streamer")
	routes, err := Router.GetAll()
	if err != nil {
		log.Errorf("Get all lenz route failed %s", err)
		return
	}
	for _, route := range routes {
		if !Router.Remove(route.ID) {
			log.Infof("Close lenz route failed %s", route.ID)
		}
	}
}
