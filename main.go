package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/HunanTV/eru-agent/api"
	"github.com/HunanTV/eru-agent/app"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/health"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/network"
	"github.com/HunanTV/eru-agent/status"
	"github.com/HunanTV/eru-agent/utils"
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

	api.Serve()
	status.Start()
	health.Check()
	app.Limit()

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGQUIT)
	logs.Info("Eru Agent Catch", <-c)
}
