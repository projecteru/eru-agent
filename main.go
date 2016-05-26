package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/projecteru/eru-agent/api"
	"github.com/projecteru/eru-agent/app"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/lenz"
	"github.com/projecteru/eru-agent/network"
	"github.com/projecteru/eru-agent/status"
	"github.com/projecteru/eru-agent/utils"

	log "github.com/Sirupsen/logrus"
)

func main() {
	g.LoadConfig()
	g.InitialConn()
	g.InitTransfers()
	defer g.CloseConn()

	lenz.InitLenz()
	status.InitStatus()
	network.InitVlan()
	defer lenz.CloseLenz()

	utils.WritePid(g.Config.PidFile)
	defer os.Remove(g.Config.PidFile)

	app.Limit()
	app.Metric()
	api.Serve()
	status.Start()

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGQUIT)
	select {
	case s := <-c:
		log.Infof("Eru Agent Catch %s", s)
		os.Exit(0)
	case e := <-g.ErrChan:
		log.Panicf("Eru Agent Error %s", e)
	}
}
