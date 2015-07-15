package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/HunanTV/eru-agent/api"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/status"
	"github.com/HunanTV/eru-agent/utils"
)

var VLan *VLanSetter

func main() {
	g.LoadConfig()
	g.InitialConn()
	defer g.Rds.Close()

	//lenz.Lenz = lenz.NewLenz()
	//metrics.Metrics = metrics.NewMetricsRecorder()
	status.InitStatus()

	utils.WritePid(g.Config.PidFile)
	defer os.Remove(g.Config.PidFile)

	//VLan = NewVLanSetter()
	//go VLan.Watcher()
	// Watch Lan first
	status.Load()
	status.StartMonitor()
	//go status.Status.Watcher()
	//go Ping()

	if g.Config.API.Http {
		go api.HTTPServe()
	}

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGQUIT)
	logs.Info("Catch", <-c)
}
